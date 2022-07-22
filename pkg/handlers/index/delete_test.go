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
	"github.com/zinclabs/zinc/test/utils"
)

func TestDelete(t *testing.T) {
	type args struct {
		code   int
		params map[string]string
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
				params: map[string]string{"target": "TestIndexDelete.index_1"},
				result: "deleted",
			},
			wantErr: false,
		},
		{
			name: "normal with wildcards",
			args: args{
				code:   http.StatusOK,
				params: map[string]string{"target": "log-*-TestIndexDelete"},
				result: "deleted",
			},
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				code:   http.StatusBadRequest,
				params: map[string]string{"target": ""},
				result: "index name cannot be empty",
			},
			wantErr: false,
		},
		{
			name: "index does not exist",
			args: args{
				code:   http.StatusBadRequest,
				params: map[string]string{"target": "Index-Not-Exists"},
				result: "index Index-Not-Exists does not exists",
			},
			wantErr: false,
		},
	}

	t.Run("prepare", func(t *testing.T) {
		prepareIndex(t, "TestIndexDelete.index_1", "disk")
		prepareIndex(t, "log-3342-44-TestIndexDelete", "disk")
		prepareIndex(t, "log-3122-44-TestIndexDelete", "disk")
		prepareIndex(t, "log-vvs323-44-TestIndexDelete", "disk")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := utils.NewGinContext()
			utils.SetGinRequestParams(c, tt.args.params)
			Delete(c)
			assert.Equal(t, tt.args.code, w.Code)
			assert.Contains(t, w.Body.String(), tt.args.result)

			resp := make(map[string]string)
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
		})
	}
}

func prepareIndex(t *testing.T, name, storageType string) {
	index, err := core.NewIndex(name, storageType, 2)
	assert.NoError(t, err)
	assert.NotNil(t, index)

	err = core.StoreIndex(index)
	assert.NoError(t, err)
}
