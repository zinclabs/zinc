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
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/test/utils"
)

func TestMapping(t *testing.T) {
	t.Run("create index", func(t *testing.T) {
		index, err := core.NewIndex("TestMapping.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("set mapping", func(t *testing.T) {
		type args struct {
			code    int
			data    map[string]interface{}
			rawData string
			target  string
			result  string
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{
				name: "normal",
				args: args{
					code: http.StatusOK,
					data: map[string]interface{}{
						"properties": map[string]interface{}{
							"Athlete": map[string]interface{}{
								"type":          "text",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  false,
								"highlightable": false,
							},
							"City": map[string]interface{}{
								"type":          "keyword",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
							"Gender": map[string]interface{}{
								"type":          "bool",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
							"time": map[string]interface{}{
								"type":          "time",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
						},
					},
					target: "TestMapping.index_1",
					result: `{"message":"ok"}`,
				},
				wantErr: false,
			},
			{
				name: "with not exists index",
				args: args{
					code: http.StatusOK,
					data: map[string]interface{}{
						"properties": map[string]interface{}{
							"Athlete": map[string]interface{}{
								"type":          "text",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  false,
								"highlightable": false,
							},
						},
					},
					target: "TestMapping.index_2",
					result: `{"message":"ok"}`,
				},
				wantErr: true,
			},
			{
				name: "empty_body",
				args: args{
					code:   http.StatusOK,
					data:   map[string]interface{}{},
					target: "TestMapping.index_3",
					result: `{"message":"ok"}`,
				},
				wantErr: false,
			},
			{
				name: "empty",
				args: args{
					code:   http.StatusBadRequest,
					data:   map[string]interface{}{},
					target: "",
					result: `{"error":"index.name should be not empty"}`,
				},
				wantErr: false,
			},
			{
				name: "with error json",
				args: args{
					code:    http.StatusBadRequest,
					rawData: `{"x":y}`,
					target:  "TestMapping.index_4",
					result:  `{"error":"invalid character 'y' looking for beginning of value"}`,
				},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c, w := utils.NewGinContext()
				if tt.args.data != nil {
					utils.SetGinRequestData(c, tt.args.data)
				}
				if tt.args.rawData != "" {
					utils.SetGinRequestData(c, tt.args.rawData)
				}
				utils.SetGinRequestParams(c, map[string]string{"target": tt.args.target})
				SetMapping(c)
				assert.Equal(t, tt.args.code, w.Code)
				assert.Equal(t, tt.args.result, w.Body.String())
			})
		}
	})

	t.Run("get mapping", func(t *testing.T) {
		type args struct {
			code   int
			target string
			result string
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{
				name: "normal",
				args: args{
					code:   http.StatusOK,
					target: "TestMapping.index_1",
					result: `{"mappings":{"properties`,
				},
				wantErr: false,
			},
			{
				name: "empty",
				args: args{
					code:   http.StatusBadRequest,
					target: "",
					result: `does not exists`,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c, w := utils.NewGinContext()
				utils.SetGinRequestParams(c, map[string]string{"target": tt.args.target})
				GetMapping(c)
				assert.Equal(t, tt.args.code, w.Code)
				assert.Contains(t, w.Body.String(), tt.args.result)
			})
		}
	})

	t.Run("delete index", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_ = core.DeleteIndex(fmt.Sprintf("TestMapping.index_%d", i))
		}
	})
}
