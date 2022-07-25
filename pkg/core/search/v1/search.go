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

package v1

import (
	"context"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search/highlight"
	"github.com/rs/zerolog/log"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/uquery/timerange"
)

func Search(index *core.Index, iQuery *ZincQuery) (*SearchResponse, error) {
	var searchRequest bluge.SearchRequest
	if iQuery.MaxResults > config.Global.MaxResults {
		iQuery.MaxResults = config.Global.MaxResults
	}

	sourceCtl := &Source{Enable: true}
	switch iQuery.Source.(type) {
	case bool:
		sourceCtl.Enable = iQuery.Source.(bool)
	case []interface{}:
		v := iQuery.Source.([]interface{})
		sourceCtl.Fields = make(map[string]bool, len(v))
		for _, field := range v {
			if fv, ok := field.(string); ok {
				sourceCtl.Fields[fv] = true
			}
		}
	}

	// highlight
	var highlighter *highlight.SimpleHighlighter
	if iQuery.Highlight != nil {
		if len(iQuery.Highlight.PreTags) > 0 && len(iQuery.Highlight.PostTags) > 0 {
			highlighter = highlight.NewHTMLHighlighterTags(iQuery.Highlight.PreTags[0], iQuery.Highlight.PostTags[0])
		} else {
			highlighter = highlight.NewHTMLHighlighter()
		}
	}

	resp := new(SearchResponse)
	resp.Hits.Hits = make([]Hit, 0)

	var err error
	switch iQuery.SearchType {
	case "alldocuments":
		searchRequest, err = AllDocuments(iQuery)
	case "wildcard":
		searchRequest, err = WildcardQuery(iQuery)
	case "fuzzy":
		searchRequest, err = FuzzyQuery(iQuery)
	case "term":
		searchRequest, err = TermQuery(iQuery)
	case "daterange":
		searchRequest, err = DateRangeQuery(iQuery)
	case "matchall":
		searchRequest, err = MatchAllQuery(iQuery)
	case "match":
		searchRequest, err = MatchQuery(iQuery)
	case "matchphrase":
		searchRequest, err = MatchPhraseQuery(iQuery)
	case "multiphrase":
		searchRequest, err = MultiPhraseQuery(iQuery)
	case "prefix":
		searchRequest, err = PrefixQuery(iQuery)
	case "querystring":
		searchRequest, err = QueryStringQuery(iQuery)
	default:
		// default use alldocuments search
		searchRequest, err = AllDocuments(iQuery)
	}

	if err != nil {
		resp.Error = err.Error()
		return resp, err
	}

	// handle aggregations
	mappings := index.GetMappings()
	err = AddAggregations(searchRequest, iQuery.Aggregations, mappings)
	if err != nil {
		resp.Error = err.Error()
		return resp, err
	}

	timeMin, timeMax := timerange.Query(iQuery.Query)
	readers, err := index.GetReaders(timeMin, timeMax)
	if err != nil {
		log.Printf("error accessing reader: %s", err.Error())
	}
	defer func() {
		for _, reader := range readers {
			reader.Close()
		}
	}()

	dmi, err := bluge.MultiSearch(context.Background(), searchRequest, readers...)
	if err != nil {
		log.Printf("error executing search: %s", err.Error())
	}

	var hits = make([]Hit, 0)

	next, err := dmi.Next()
	for err == nil && next != nil {
		var result map[string]interface{}
		var id string
		var timestamp time.Time
		var highlightData map[string]interface{}
		if iQuery.Highlight != nil {
			highlightData = make(map[string]interface{})
		}
		err = next.VisitStoredFields(func(field string, value []byte) bool {
			switch field {
			case "_id":
				id = string(value)
			case "@timestamp":
				timestamp, _ = bluge.DecodeDateTime(value)
			case "_source":
				result = HandleSource(sourceCtl, value)
			default:
				// highlight
				if iQuery.Highlight != nil && iQuery.Highlight.Fields != nil {
					if options, ok := iQuery.Highlight.Fields[field]; ok {
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
			log.Printf("error accessing stored fields: %s", err.Error())
		}

		hit := Hit{
			Index:     index.GetName(),
			Type:      "_doc",
			ID:        id,
			Score:     next.Score,
			Timestamp: timestamp,
			Source:    result,
			Highlight: highlightData,
		}
		hits = append(hits, hit)

		next, err = dmi.Next()
	}
	if err != nil {
		log.Printf("error iterating results: %s", err.Error())
	}

	resp.Took = int(dmi.Aggregations().Duration().Milliseconds())
	resp.Hits.Total.Value = int(dmi.Aggregations().Count())
	resp.Hits.MaxScore = dmi.Aggregations().Metric("max_score")
	resp.Hits.Hits = hits

	if len(iQuery.Aggregations) > 0 {
		resp.Aggregations, err = ParseAggregations(dmi.Aggregations())
		if err != nil {
			log.Printf("error parse aggregation results: %s", err.Error())
		}
		if len(resp.Aggregations) > 0 {
			delete(resp.Aggregations, "count")
			delete(resp.Aggregations, "duration")
			delete(resp.Aggregations, "max_score")
		}
	}

	return resp, nil
}
