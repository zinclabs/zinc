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

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/meta"
)

func TestMultiSearch(t *testing.T) {
	type args struct {
		indexName string
		query     *meta.ZincQuery
	}
	tests := []struct {
		name    string
		args    args
		want    *meta.SearchResponse
		wantErr bool
	}{
		{
			name: "multiple search",
			args: args{
				indexName: "",
				query: &meta.ZincQuery{
					Query: &meta.Query{
						MatchAll: &meta.MatchAllQuery{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "wildcard search",
			args: args{
				indexName: "TestMultiSearch.*",
				query: &meta.ZincQuery{
					Query: &meta.Query{
						MatchAll: &meta.MatchAllQuery{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				indexName: "TestMultiSearchNotExist",
				query: &meta.ZincQuery{
					Query: &meta.Query{
						MatchAll: &meta.MatchAllQuery{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "timeout",
			args: args{
				indexName: "",
				query: &meta.ZincQuery{
					Query: &meta.Query{
						MatchAll: &meta.MatchAllQuery{},
					},
					Timeout: 1,
				},
			},
			wantErr: false,
		},
	}

	indexNames := []string{"TestMultiSearch.index_1", "TestMultiSearch.index_2"}
	t.Run("prepare", func(t *testing.T) {
		for _, indexName := range indexNames {
			index, err := NewIndex(indexName, "disk", 2)
			assert.NoError(t, err)
			assert.NotNil(t, index)

			err = StoreIndex(index)
			assert.NoError(t, err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var got *meta.SearchResponse
			if tt.args.indexName == "" {
				got, err = MultiSearch(nil, tt.args.query)
			} else {
				got, err = MultiSearch([]string{tt.args.indexName}, tt.args.query)
			}
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		for _, indexName := range indexNames {
			err := DeleteIndex(indexName)
			assert.NoError(t, err)
		}
	})
}

func TestIsMatchIndex(t *testing.T) {
	ret := isMatchIndex("abc", "a") //  false
	assert.False(t, ret)
	ret = isMatchIndex("abc", "a*") // true
	assert.True(t, ret)
	ret = isMatchIndex("abc", "*bc") // true
	assert.True(t, ret)
	ret = isMatchIndex("abc", "bc") // false
	assert.False(t, ret)
	ret = isMatchIndex("abc", "abc") // true
	assert.True(t, ret)
}
