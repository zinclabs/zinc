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
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/test/utils"
)

func TestList(t *testing.T) {
	t.Run("prepare", func(t *testing.T) {
		index, err := core.NewIndex("TestIndexList.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("list", func(t *testing.T) {
		sortBy := []string{"name", "doc_num", "shard_num", "storage_size", "storage_type", "wal_size"}
		descArr := []string{"false", "true"}
		for _, s := range sortBy {
			for _, d := range descArr {

				c, w := utils.NewGinContext()
				params := map[string]string{
					"page_num":  "1",
					"page_size": "20",
					"sort_by":   s,
					"desc":      d,
					"name":      "TestIndexList.index_1",
				}
				utils.SetGinRequestURL(c, "/api/index", params)
				List(c)
				assert.Equal(t, http.StatusOK, w.Code)
				assert.NotNil(t, w.Body)

				resp := struct {
					List []*meta.Index `json:"list"`
					Page meta.Page     `json:"page"`
				}{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotNil(t, resp.List)
				assert.NotNil(t, resp.Page)
				assert.Equal(t, len(resp.List), 1)
				assert.Equal(t, resp.List[0].Name, "TestIndexList.index_1")
				assert.Equal(t, resp.Page.PageSize, int64(20))
				assert.Equal(t, resp.Page.PageNum, int64(1))
				assert.Equal(t, resp.Page.Total, int64(1))
				assert.Equal(t, len(resp.List), 1)
			}
		}
	})

	t.Run("cleanup", func(t *testing.T) {
		err := core.DeleteIndex("TestIndexList.index_1")
		assert.NoError(t, err)
	})
}

func TestIndexNameList(t *testing.T) {
	t.Run("prepare", func(t *testing.T) {
		index, err := core.NewIndex("TestIndexNameList.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("indexNameList", func(t *testing.T) {
		c, w := utils.NewGinContext()
		params := map[string]string{
			"name": "TestIndexNameList.index_1",
		}
		utils.SetGinRequestURL(c, "/api/index_name", params)
		IndexNameList(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotNil(t, w.Body)
		var resp []string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, len(resp), 1)
		assert.Equal(t, resp[0], "TestIndexNameList.index_1")
		assert.Contains(t, w.Body.String(), "TestIndexNameList.index_1")
	})

	t.Run("cleanup", func(t *testing.T) {
		err := core.DeleteIndex("TestIndexNameList.index_1")
		assert.NoError(t, err)
	})
}
