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

package search

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// SearchDSL searches the index for the given http request from end user

// @Id Search
// @Summary Search V2 DSL for compatible ES
// @Tags    Search
// @Accept  json
// @Produce json
// @Param   index  path  string  true  "Index"
// @Param   query  body  meta.ZincQueryForSDK true  "Query"
// @Success 200 {object} meta.SearchResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Router /es/{index}/_search [post]
func SearchDSL(c *gin.Context) {
	indexName := c.Param("target")

	query := &meta.ZincQuery{Size: 10}
	if err := zutils.GinBindJSON(c, query); err != nil {
		log.Printf("handlers.search.searchDSL: %s", err.Error())
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	resp, err := searchIndex(strings.Split(indexName, ","), query)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	if indexName != "" {
		idx, _ := core.ZINC_INDEX_LIST.Get(indexName)
		storageSize := idx.GetStats().StorageSize
		eventData := make(map[string]interface{})
		eventData["search_type"] = "query_dsl"
		eventData["search_index_storage"] = idx.GetStorageType()
		eventData["search_index_size_in_mb"] = storageSize / 1024 / 1024
		eventData["time_taken_to_search_in_ms"] = resp.Took
		eventData["aggregations_count"] = len(query.Aggregations)
		core.Telemetry.Event("search", eventData)
	}

	c.JSON(http.StatusOK, resp)
}

// MultipleSearch like bulk searches

// @Id MSearch
// @Summary Search V2 MultipleSearch for compatible ES
// @Tags    Search
// @Accept  plain
// @Produce json
// @Param   query  body  string  true  "Query"
// @Success 200 {object} meta.SearchResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Router /es/_msearch [post]
func MultipleSearch(c *gin.Context) {
	indexName := c.Param("target")
	defaultIndexNames := make([]string, 0)
	if indexName != "" {
		defaultIndexNames = strings.Split(indexName, ",")
	}

	responses := make([]interface{}, 0)

	// Prepare to read the entire raw text of the body
	scanner := bufio.NewScanner(c.Request.Body)
	defer c.Request.Body.Close()

	const maxCapacityPerLine = 1024 * 1024
	buf := make([]byte, maxCapacityPerLine)
	scanner.Buffer(buf, maxCapacityPerLine)

	indexNames := make([]string, 0)
	nextLineIsData := false

	var doc map[string]interface{}
	var err error
	for scanner.Scan() { // Read each line
		if nextLineIsData {
			nextLineIsData = false
			query := &meta.ZincQuery{Size: 10}
			if err = json.Unmarshal(scanner.Bytes(), &query); err != nil {
				log.Error().Msgf("handlers.search.MultipleSearch.json.Unmarshal: %s, err %s", scanner.Text(), err.Error())
				responses = append(responses, &meta.SearchResponse{Error: err.Error()})
				continue
			}
			// search query
			resp, err := searchIndex(indexNames, query)
			if err != nil {
				log.Error().Msgf("handlers.search.MultipleSearch.searchIndex: err %s", err.Error())
				responses = append(responses, &meta.SearchResponse{Error: err.Error()})
			} else {
				responses = append(responses, resp)
			}
		} else {
			nextLineIsData = true
			indexNames = indexNames[:0]
			if err = json.Unmarshal(scanner.Bytes(), &doc); err != nil {
				log.Error().Msgf("handlers.search.MultipleSearch.json.Unmarshal: %s, err %s", scanner.Text(), err.Error())
				continue
			}
			if v, ok := doc["index"]; ok {
				switch v := v.(type) {
				case string:
					indexNames = append(indexNames, v)
				case []interface{}:
					for _, v := range v {
						indexNames = append(indexNames, v.(string))
					}
				}
			} else {
				indexNames = append(indexNames, defaultIndexNames...)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"responses": responses})
}

func searchIndex(indexNames []string, query *meta.ZincQuery) (*meta.SearchResponse, error) {
	var indexName = ""
	if len(indexNames) > 0 {
		indexName = indexNames[0]
	}
	var err error
	var resp *meta.SearchResponse
	if indexName == "" || strings.HasSuffix(indexName, "*") || strings.HasPrefix(indexName, "*") || len(indexNames) > 1 {
		resp, err = core.MultiSearch(indexNames, query)
	} else {
		index, exists := core.GetIndex(indexName)
		if !exists {
			return nil, fmt.Errorf("index %s does not exists", indexName)
		}
		resp, err = index.Search(query)
	}
	return resp, err
}
