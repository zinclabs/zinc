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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/test/utils"
)

func TestSearchV1(t *testing.T) {
	indexName := "TestSearchV1.index_1"
	type args struct {
		code   int
		data   string
		params map[string]string
		result string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "normal",
			args: args{
				code:   http.StatusOK,
				data:   `{"query_type":"match_all","max_results":10}`,
				params: map[string]string{"target": indexName},
				result: "{\"total\":",
			},
		},
		{
			name: "index not found",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"query_type":"match_all","max_results":10}`,
				params: map[string]string{"target": "NotExist" + indexName},
				result: "does not exists",
			},
		},
		{
			name: "query jsone error",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"query_type":"match_all","max_results":10,{x}}`,
				params: map[string]string{"target": indexName},
				result: "invalid character",
			},
		},
	}

	t.Run("prepare", func(t *testing.T) {
		index, err := core.NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := utils.NewGinContext()
			utils.SetGinRequestData(c, tt.args.data)
			utils.SetGinRequestParams(c, tt.args.params)
			SearchV1(c)
			assert.Equal(t, tt.args.code, w.Code)
			assert.Contains(t, w.Body.String(), tt.args.result)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err := core.DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}
