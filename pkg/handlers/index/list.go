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
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
)

// @Id ListIndexes
// @Summary List indexes
// @Tags    Index
// @Param   page_num  query  integer  false  "page num"
// @Param   page_size  query  integer  false  "page size"
// @Param   sort_by  query  string  false  "sort by"
// @Param   desc  query  boolen  false  "desc"
// @Param   name  query  string  false  "name"
// @Produce json
// @Success 200 {object} {list:[]core.Index, page: meta.Page}
// @Router /api/index [get]
func List(c *gin.Context) {
	page := meta.NewPage(c)
	sortBy := c.DefaultQuery("sort_by", "name")
	desc, _ := strconv.ParseBool(c.DefaultQuery("desc", "false"))
	name := c.DefaultQuery("name", "")

	items := core.ZINC_INDEX_LIST.ListStat()

	if len(name) > 0 {
		var res []*core.Index
		for _, item := range items {
			if strings.Contains(item.GetName(), name) {
				res = append(res, item)
			}
		}
		items = res
	}

	switch sortBy {
	case "doc_num":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GetStats().DocNum > items[j].GetStats().DocNum
			} else {
				return items[i].GetStats().DocNum < items[j].GetStats().DocNum
			}
		})
	case "shard_num":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GetShardNum() > items[j].GetShardNum()
			} else {
				return items[i].GetShardNum() < items[j].GetShardNum()
			}
		})
	case "storage_size":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GetStats().StorageSize > items[j].GetStats().StorageSize
			} else {
				return items[i].GetStats().StorageSize < items[j].GetStats().StorageSize
			}
		})
	case "storage_type":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GetStorageType() > items[j].GetStorageType()
			} else {
				return items[i].GetStorageType() < items[j].GetStorageType()
			}
		})
	case "wal_size":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GetWALSize() > items[j].GetWALSize()
			} else {
				return items[i].GetWALSize() < items[j].GetWALSize()
			}
		})
	case "name":
		fallthrough
	default:
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GetName() > items[j].GetName()
			} else {
				return items[i].GetName() < items[j].GetName()
			}
		})
	}

	page.Total = int64(len(items))
	startIndex, endIndex := page.GetStartEndIndex()
	if endIndex > 0 {
		items = items[startIndex:endIndex]
	} else {
		items = []*core.Index{}
	}

	c.JSON(http.StatusOK, gin.H{
		"list": items,
		"page": page,
	})
}

// @Id IndexNameList
// @Summary List index Name
// @Tags    Index
// @Param   name  query  string  false  "IndexName"
// @Produce json
// @Success 200 {object} []string
// @Router /api/index_name [get]
func IndexNameList(c *gin.Context) {
	queryName := strings.ToLower(c.DefaultQuery("name", ""))
	var items []string
	names := core.ZINC_INDEX_LIST.ListName()
	if queryName == "" {
		items = names
	} else {
		for _, name := range names {
			if strings.Contains(strings.ToLower(name), queryName) {
				items = append(items, name)
			}
		}
	}

	count := 30
	if len(items) > count {
		c.JSON(http.StatusOK, items[0:count])
	} else {
		c.JSON(http.StatusOK, items)
	}
}
