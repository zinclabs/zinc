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

package index

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
	zincanalysis "github.com/zinclabs/zinc/pkg/uquery/analysis"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// @Id GetSettings
// @Summary Get index settings
// @Tags    Index
// @Produce json
// @Param   index path  string  true  "Index"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} meta.HTTPResponseError
// @Router /api/{index}/_settings [get]
func GetSettings(c *gin.Context) {
	indexName := c.Param("target")
	index, exists := core.GetIndex(indexName)
	if !exists {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index " + indexName + " does not exists"})
		return
	}

	settings := index.GetSettings()
	if settings == nil {
		settings = new(meta.IndexSettings)
	}

	c.JSON(http.StatusOK, gin.H{index.GetName(): gin.H{"settings": settings}})
}

// @Id SetSettings
// @Summary Set index Settings
// @Tags    Index
// @Accept  json
// @Produce json
// @Param   index    path  string             true  "Index"
// @Param   settings body  meta.IndexSettings true  "Settings"
// @Success 200 {object} meta.HTTPResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Failure 500 {object} meta.HTTPResponseError
// @Router /api/{index}/_settings [put]
func SetSettings(c *gin.Context) {
	indexName := c.Param("target")
	if indexName == "" {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index.name should be not empty"})
		return
	}

	var settings *meta.IndexSettings
	if err := zutils.GinBindJSON(c, &settings); err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	if settings == nil {
		c.JSON(http.StatusOK, meta.HTTPResponse{Message: "ok"})
		return
	}

	analyzers, err := zincanalysis.RequestAnalyzer(settings.Analysis)
	if err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	shardsNum := config.Global.Shard.Num
	if settings.NumberOfShards != 0 {
		shardsNum = settings.NumberOfShards
	}
	index, exists, err := core.GetOrCreateIndex(indexName, "", shardsNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
		return
	}
	if exists {
		// it can only change settings.NumberOfReplicas when index exists
		if settings.NumberOfReplicas > 0 {
			indexSettings := index.GetSettings()
			atomic.StoreInt64(&indexSettings.NumberOfReplicas, settings.NumberOfReplicas)
		}
		if settings.Analysis != nil && len(settings.Analysis.Analyzer) > 0 {
			c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "can't update analyzer for existing index"})
			return
		}
		// store index
		if err := core.StoreIndex(index); err != nil {
			c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, meta.HTTPResponse{Message: "ok"})
		return
	}

	// update settings
	_ = index.SetSettings(settings)

	// update analyzers
	_ = index.SetAnalyzers(analyzers)

	// store index
	if err := core.StoreIndex(index); err != nil {
		c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, meta.HTTPResponse{Message: "ok"})
}
