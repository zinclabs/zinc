/* Copyright 2022 Zinc Labs Inc. and Contributors
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package core

import (
	"context"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/blugelabs/bluge/search/highlight"
	"github.com/rs/zerolog/log"

	zsearch "github.com/zinclabs/zinc/pkg/bluge/search"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/uquery"
	"github.com/zinclabs/zinc/pkg/uquery/fields"
	"github.com/zinclabs/zinc/pkg/uquery/source"
	"github.com/zinclabs/zinc/pkg/uquery/timerange"
)

func (index *Index) Search(query *meta.ZincQuery) (*meta.SearchResponse, error) {
	mappings := index.GetMappings()
	analyzers := index.GetAnalyzers()
	searchRequest, err := uquery.ParseQueryDSL(query, mappings, analyzers)
	if err != nil {
		return nil, err
	}

	timeMin, timeMax := timerange.Query(query.Query)
	readers, err := index.GetReaders(timeMin, timeMax)
	if err != nil {
		log.Printf("index.SearchV2: error accessing reader: %s", err.Error())
		return nil, err
	}
	defer func() {
		for _, reader := range readers {
			reader.Close()
		}
	}()

	ctx := context.Background()
	var cancel context.CancelFunc
	if query.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(query.Timeout)*time.Second)
		defer cancel()
	}

	// dmi, err := bluge.MultiSearch(ctx, searchRequest, readers...)
	dmi, err := zsearch.MultiSearch(ctx, searchRequest, readers...)
	if err != nil {
		log.Printf("index.SearchV2: error executing search: %s", err.Error())
		if err == context.DeadlineExceeded {
			return &meta.SearchResponse{
				TimedOut: true,
				Error:    err.Error(),
				Hits:     meta.Hits{Hits: []meta.Hit{}},
			}, nil
		}
		return nil, err
	}

	return searchV2(index.GetShardNum(), int64(len(readers)), dmi, query, mappings)
}

func searchV2(shardNum, readerNum int64, dmi search.DocumentMatchIterator, query *meta.ZincQuery, mappings *meta.Mappings) (*meta.SearchResponse, error) {
	resp := &meta.SearchResponse{
		Hits: meta.Hits{Hits: []meta.Hit{}},
	}

	// highlight
	var highlighter *highlight.SimpleHighlighter
	if query.Highlight != nil {
		if len(query.Highlight.PreTags) > 0 && len(query.Highlight.PostTags) > 0 {
			highlighter = highlight.NewHTMLHighlighterTags(query.Highlight.PreTags[0], query.Highlight.PostTags[0])
		} else {
			highlighter = highlight.NewHTMLHighlighter()
		}
	}

	Hits := make([]meta.Hit, 0)
	next, err := dmi.Next()
	for err == nil && next != nil {
		var id string
		var indexName string
		var timestamp time.Time
		var sourceData map[string]interface{}
		var fieldsData map[string]interface{}
		var highlightData map[string]interface{}
		if query.Highlight != nil {
			highlightData = make(map[string]interface{})
		}
		err = next.VisitStoredFields(func(field string, value []byte) bool {
			switch field {
			case "_id":
				id = string(value)
			case "_index":
				indexName = string(value)
			case "@timestamp":
				timestamp, _ = bluge.DecodeDateTime(value)
			case "_source":
				sourceData = source.Response(query.Source.(*meta.Source), value)
				if query.Fields != nil {
					fieldsData = fields.Response(query.Fields.([]*meta.Field), value, mappings)
				}
			default:
				// highlight
				if query.Highlight != nil && query.Highlight.Fields != nil {
					if options, ok := query.Highlight.Fields[field]; ok {
						if v, ok := next.Locations[field]; ok {
							if len(options.PreTags) > 0 && len(options.PostTags) > 0 {
								highlighter := highlight.NewHTMLHighlighterTags(options.PreTags[0], options.PostTags[0])
								highlightData[field] = highlighter.BestFragments(v, value, options.NumberOfFragments)
							} else {
								highlightData[field] = highlighter.BestFragments(v, value, options.NumberOfFragments)
							}
						}
					}
				}
			}

			return true
		})
		if err != nil {
			log.Printf("core.SearchV2: error accessing stored fields: %s", err.Error())
			continue
		}

		hit := meta.Hit{
			Index:     indexName,
			Type:      "_doc",
			ID:        id,
			Score:     next.Score,
			Timestamp: timestamp,
			Source:    sourceData,
			Fields:    fieldsData,
			Highlight: highlightData,
		}
		Hits = append(Hits, hit)

		next, err = dmi.Next()
	}
	if err != nil {
		log.Printf("core.SearchV2: error iterating results: %s", err.Error())
	}

	resp.Took = int(dmi.Aggregations().Duration().Milliseconds())
	resp.Shards = meta.Shards{Total: shardNum, Successful: readerNum, Skipped: shardNum - readerNum}
	resp.Hits = meta.Hits{
		Total:    meta.Total{Value: int(dmi.Aggregations().Count())},
		MaxScore: dmi.Aggregations().Metric("max_score"),
		Hits:     Hits,
	}

	if err := uquery.FormatResponse(resp, query, dmi.Aggregations()); err != nil {
		log.Printf("core.SearchV2: error format response: %s", err.Error())
	}

	return resp, nil
}
