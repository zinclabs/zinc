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
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/core"
	v1 "github.com/zinclabs/zinc/pkg/core/search/v1"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// SearchV1 searches the index for the given http request from end user

// @Id SearchV1
// @Summary Search V1
// @Tags    Search
// @Accept  json
// @Produce json
// @Param   index  path  string  true  "Index"
// @Param   query  body  v1.ZincQueryForSDK  true  "Query"
// @Success 200 {object} v1.SearchResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Router /api/{index}/_search [post]
func SearchV1(c *gin.Context) {
	indexName := c.Param("target")
	index, exists := core.GetIndex(indexName)
	if !exists {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index " + indexName + " does not exists"})
		return
	}

	var iQuery v1.ZincQuery
	iQuery.MaxResults = 10
	if err := zutils.GinBindJSON(c, &iQuery); err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	res, err := v1.Search(index, &iQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	storageSize := index.GetStats().StorageSize
	eventData := make(map[string]interface{})
	eventData["search_type"] = iQuery.SearchType
	eventData["search_index_storage"] = index.GetStorageType()
	eventData["search_index_size_in_mb"] = storageSize / 1024 / 1024
	eventData["time_taken_to_search_in_ms"] = res.Took
	eventData["aggregations_count"] = len(iQuery.Aggregations)
	core.Telemetry.Event("search", eventData)

	c.JSON(http.StatusOK, res)
}
