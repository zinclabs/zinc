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

	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/uquery/mappings"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// @Id GetMapping
// @Summary Get index mappings
// @Tags    Index
// @Produce json
// @Param   index  path  string  true  "Index"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} meta.HTTPResponseError
// @Router /api/{index}/_mapping [get]
func GetMapping(c *gin.Context) {
	indexName := c.Param("target")
	index, exists := core.GetIndex(indexName)
	if !exists {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index " + indexName + " does not exists"})
		return
	}

	// format mappings
	mappings := index.GetMappings()

	c.JSON(http.StatusOK, gin.H{index.GetName(): gin.H{"mappings": mappings}})
}

// @Id SetMapping
// @Summary Set index mappings
// @Tags    Index
// @Accept  json
// @Produce json
// @Param   index   path  string        true  "Index"
// @Param   mapping body  meta.Mappings true  "Mapping"
// @Success 200 {object} meta.HTTPResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Failure 500 {object} meta.HTTPResponseError
// @Router /api/{index}/_mapping [put]
func SetMapping(c *gin.Context) {
	indexName := c.Param("target")
	if indexName == "" {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index.name should be not empty"})
		return
	}

	var mappingRequest map[string]interface{}
	if err := zutils.GinBindJSON(c, &mappingRequest); err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	mappings, err := mappings.Request(nil, mappingRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	index, exists, err := core.GetOrCreateIndex(indexName, "", 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	// check if mapping field is exists
	if exists {
		indexMappings := index.GetMappings()
		if indexMappings != nil && indexMappings.Len() > 0 {
			for field := range mappings.ListProperty() {
				if _, ok := indexMappings.GetProperty(field); ok {
					c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index [" + indexName + "] already exists mapping of field [" + field + "]"})
					return
				}
			}
		}
		// add mappings
		for field, prop := range mappings.ListProperty() {
			indexMappings.SetProperty(field, prop)
		}
		mappings = indexMappings
	}

	// update mappings
	if mappings != nil && mappings.Len() > 0 {
		for k, v := range mappings.Properties {
			if v.Fields == nil {
				continue
			}

			update := false
			for kField, field := range v.Fields {
				if field.Fields != nil {
					field.Fields = nil
					v.Fields[kField] = field
					update = true
				}
			}

			if update {
				mappings.Properties[k] = v
			}
		}
		_ = index.SetMappings(mappings)
	}

	// store index
	if err := core.StoreIndex(index); err != nil {
		c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, meta.HTTPResponse{Message: "ok"})
}
