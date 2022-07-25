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
	"fmt"
	"strings"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/analysis"
	"github.com/rs/zerolog/log"

	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/uquery"
	"github.com/zinclabs/zinc/pkg/uquery/timerange"
)

func MultiSearch(indexNames []string, query *meta.ZincQuery) (*meta.SearchResponse, error) {
	var mappings *meta.Mappings
	var analyzers map[string]*analysis.Analyzer
	var readers []*bluge.Reader
	var shardNum int64

	timeMin, timeMax := timerange.Query(query.Query)
	isMatched := false
	hasIndex := false
	for _, index := range ZINC_INDEX_LIST.List() {
		if len(indexNames) > 0 {
			for _, indexName := range indexNames {
				isMatched = isMatchIndex(index.GetName(), indexName)
				if isMatched {
					hasIndex = true
					break
				}
			}
			if !isMatched {
				continue
			}
		}

		reader, err := index.GetReaders(timeMin, timeMax)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader...)
		shardNum += index.GetShardNum()
		if mappings == nil {
			mappings = index.GetMappings()
			analyzers = index.GetAnalyzers()
		}

	}

	if len(readers) == 0 {
		if !hasIndex {
			return nil, fmt.Errorf("core.MultiSearchV2: error accessing reader: no index found")
		}
		return &meta.SearchResponse{}, nil
	}

	defer func() {
		for _, reader := range readers {
			reader.Close()
		}
	}()

	searchRequest, err := uquery.ParseQueryDSL(query, mappings, analyzers)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if query.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(query.Timeout)*time.Second)
		defer cancel()
	}

	dmi, err := bluge.MultiSearch(ctx, searchRequest, readers...)
	if err != nil {
		log.Printf("core.MultiSearchV2: error executing search: %s", err.Error())
		if err == context.DeadlineExceeded {
			return &meta.SearchResponse{
				TimedOut: true,
				Error:    err.Error(),
				Hits:     meta.Hits{Hits: []meta.Hit{}},
			}, nil
		}
		return nil, err
	}

	return searchV2(shardNum, int64(len(readers)), dmi, query, mappings)
}

// isMatchIndex("abc", "a")  false
// isMatchIndex("abc", "a*") true
// isMatchIndex("abc", "*bc") true
// isMatchIndex("abc", "bc") false
// isMatchIndex("abc", "abc") true
func isMatchIndex(zincIndexName, indexName string) bool {
	if indexName == "" {
		return true
	}

	// eg.: *-test
	if strings.HasPrefix(indexName, "*") {
		return strings.HasSuffix(zincIndexName, indexName[1:])
	}

	// eg.: test-*
	if strings.HasSuffix(indexName, "*") {
		return strings.HasPrefix(zincIndexName, indexName[:len(indexName)-1])
	}

	return zincIndexName == indexName
}
