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
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
)

// Delete deletes a zinc index and its associated data. Be careful using thus as you ca't undo this action.

// @Id DeleteIndex
// @Summary Delete index
// @Tags    Index
// @Produce json
// @Param   index  path  string  true  "Index"
// @Success 200 {object} meta.HTTPResponseIndex
// @Failure 400 {object} meta.HTTPResponseError
// @Failure 500 {object} meta.HTTPResponseError
// @Router /api/index/{index} [delete]
func Delete(c *gin.Context) {
	indexNames := c.Param("target")
	if indexNames == "" {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index name cannot be empty"})
		return
	}

	indexList := core.ZINC_INDEX_LIST.List()

	for _, indexName := range strings.Split(indexNames, ",") {
		if strings.Contains(indexName, "*") { // check for wildcard
			err := deleteIndexWithWildcard(indexName, indexList)
			if err != nil {
				c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
				return
			}
			continue
		}
		if err := core.DeleteIndex(indexName); err != nil {
			c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, meta.HTTPResponse{
		Message: "deleted",
	})
}

func deleteIndexWithWildcard(indexName string, indexList []*core.Index) error {
	parts := strings.Split(indexName, "*")
	pattern := ""
	for i, part := range parts {
		pattern += part
		if i < len(parts)-1 {
			pattern += "[[:ascii:]]+"
		}
	}

	p, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	for _, i := range indexList {
		if p.MatchString(i.GetName()) {
			if err := core.DeleteIndex(i.GetName()); err != nil {
				return err
			}
		}
	}

	return nil
}
