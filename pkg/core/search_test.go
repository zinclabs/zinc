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
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/meta"
)

func TestIndex_Search(t *testing.T) {
	type args struct {
		iQuery *meta.ZincQuery
	}
	tests := []struct {
		name    string
		args    args
		want    *meta.SearchResponse
		wantErr bool
	}{
		{
			name: "Search Query - Match",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Match: map[string]*meta.MatchQuery{
							"_all": {
								Query: "Prabhat",
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - Term",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Term: map[string]*meta.TermQuery{
							"_all": {
								Value: "angeles",
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - MatchAll",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						MatchAll: &meta.MatchAllQuery{},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - wildcard",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Wildcard: map[string]*meta.WildcardQuery{
							"_all": {
								Value: "san*",
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - fuzzy",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Fuzzy: map[string]*meta.FuzzyQuery{
							"_all": {
								Value: "fransisco", // note the wrong spelling
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - fuzzy fuzziness AUTO",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Fuzzy: map[string]*meta.FuzzyQuery{
							"_all": {
								Value:     "fransisco", // note the wrong spelling,
								Fuzziness: "AUTO",
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - fuzzy fuzziness AUTO",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Fuzzy: map[string]*meta.FuzzyQuery{
							"_all": {
								Value:     "fransisco", // note the wrong spelling,
								Fuzziness: "AUTO:3,6",
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - fuzzy fuzziness 2",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						Fuzzy: map[string]*meta.FuzzyQuery{
							"_all": {
								Value:     "fransisco", // note the wrong spelling,
								Fuzziness: 2,
							},
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - querystring",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						QueryString: &meta.QueryStringQuery{
							Query: "angeles",
						},
					},
					Size: 10,
				},
			},
		},
		{
			name: "Search Query - highlight",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						QueryString: &meta.QueryStringQuery{
							Query: "angeles",
						},
					},
					Timeout: 1,
					Size:    10,
					Fields:  []interface{}{"address.city"},
					Highlight: &meta.Highlight{
						Fields: map[string]*meta.Highlight{
							"address.city": {
								PreTags:  []string{"<b>"},
								PostTags: []string{"</b>"},
							},
						},
					},
				},
			},
		},
		{
			name: "Search Query - aggs",
			args: args{
				iQuery: &meta.ZincQuery{
					Query: &meta.Query{
						MatchAll: &meta.MatchAllQuery{},
					},
					Timeout: 1,
					Size:    0,
					Aggregations: map[string]meta.Aggregations{
						"hobby": {
							Terms: &meta.AggregationsTerms{
								Field: "hobby",
							},
						},
					},
				},
			},
		},
	}

	prepareData := []map[string]interface{}{
		{
			"name": "Prabhat Sharma",
			"address": map[string]interface{}{
				"city":  "San Francisco",
				"state": "California",
			},
			"hobby": "chess",
		},
		{
			"name": "Leonardo DiCaprio",
			"address": map[string]interface{}{
				"city":  "Los angeles",
				"state": "California",
			},
			"hobby": "chess",
		},
	}

	var err error
	var index *Index
	indexName := "Search.v2.index_1"
	t.Run("Prepare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		err = StoreIndex(index)
		assert.NoError(t, err)

		index.GetMappings().SetProperty("address.city", meta.Property{
			Type:          "text",
			Index:         true,
			Store:         true,
			Highlightable: true,
		})

		for _, d := range prepareData {
			rand.Seed(time.Now().UnixNano())
			docId := rand.Intn(1000)
			err := index.CreateDocument(strconv.Itoa(docId), d, false)
			assert.NoError(t, err)
		}

		// wait for WAL write to index
		time.Sleep(time.Second)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := index.Search(tt.args.iQuery)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, got.Hits.Total.Value, 1)
		})
	}

	t.Run("Cleanup", func(t *testing.T) {
		err = DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}
