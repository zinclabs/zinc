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

package document

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// @Id UpdateDocument
// @Summary Update document with id
// @Tags    Document
// @Accept  json
// @Produce json
// @Param   index  path  string  true  "Index"
// @Param   id     path  string  true  "ID"
// @Param   document  body  map[string]interface{}  true  "Document"
// @Success 200 {object} meta.HTTPResponseID
// @Failure 400 {object} meta.HTTPResponseError
// @Failure 500 {object} meta.HTTPResponseError
// @Router /api/{index}/_update/{id} [post]
func Update(c *gin.Context) {
	indexName := c.Param("target")
	docID := c.Param("id")
	insert := c.Query("insert") // true or false
	insertBool, _ := zutils.ToBool(insert)

	var err error
	var doc map[string]interface{}
	if err = zutils.GinBindJSON(c, &doc); err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	// If id field is present then use it, else create a new UUID and use it
	if id, ok := doc["_id"]; ok {
		docID = id.(string)
	}
	if docID == "" {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "id is empty"})
		return
	}

	// If the index does not exist, then create it
	index, _, err := core.GetOrCreateIndex(indexName, "", 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	err = index.UpdateDocument(docID, doc, insertBool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, meta.HTTPResponseError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, meta.HTTPResponseID{Message: "ok", ID: docID})
}
