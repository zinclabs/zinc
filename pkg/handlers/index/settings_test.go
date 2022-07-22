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

func TestSettings(t *testing.T) {
	t.Run("create index", func(t *testing.T) {
		index, err := core.NewIndex("TestSettings.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("set settings", func(t *testing.T) {
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
						"settings": map[string]interface{}{
							"number_of_shards":   3,
							"number_of_replicas": 1,
						},
					},
					target: "TestSettings.index_1",
					result: `{"message":"ok"}`,
				},
				wantErr: false,
			},
			{
				name: "with not exists index",
				args: args{
					code: http.StatusOK,
					data: map[string]interface{}{
						"settings": map[string]interface{}{
							"number_of_shards":   3,
							"number_of_replicas": 1,
							"analysis": map[string]interface{}{
								"analyzer": map[string]interface{}{
									"default": map[string]interface{}{
										"type": "standard",
									},
								},
							},
						},
					},
					target: "TestSettings.index_2",
					result: `{"message":"ok"}`,
				},
				wantErr: true,
			},
			{
				name: "empty",
				args: args{
					code:   http.StatusOK,
					data:   map[string]interface{}{},
					target: "TestSettings.index_3",
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
					target:  "TestSettings.index_4",
					result:  `{"error":"json: null unexpected end of JSON input"}`,
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
				SetSettings(c)
				assert.Equal(t, tt.args.code, w.Code)
				assert.Equal(t, tt.args.result, w.Body.String())
			})
		}
	})

	t.Run("get settings", func(t *testing.T) {
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
					target: "TestSettings.index_1",
					result: `{"settings":`,
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
				GetSettings(c)
				assert.Equal(t, tt.args.code, w.Code)
				assert.Contains(t, w.Body.String(), tt.args.result)
			})
		}
	})

	t.Run("delete index", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_ = core.DeleteIndex(fmt.Sprintf("TestSettings.index_%d", i))
		}
	})
}
