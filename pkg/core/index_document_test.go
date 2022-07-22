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
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/bluge/aggregation"
	"github.com/zinclabs/zinc/pkg/meta"
)

func TestIndex_CreateUpdateDocument(t *testing.T) {
	type fields struct {
		Name string
	}
	type args struct {
		docID  string
		doc    map[string]interface{}
		update bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Document with generated ID",
			args: args{
				doc: map[string]interface{}{
					"name": "Hello",
					"test": "bool",
				},
				update: false,
			},
		},
		{
			name: "Document with provided ID",
			args: args{
				docID: "test1",
				doc: map[string]interface{}{
					"name": "Hello",
					"test": "Hello",
					"attr": []interface{}{
						"test",
						"test2",
					},
				},
				update: false,
			},
		},
		{
			name: "Document with type conflict",
			args: args{
				docID: "test1",
				doc: map[string]interface{}{
					"name": "Hello",
					"test": true,
				},
				update: true,
			},
			wantErr: false,
		},
		{
			name: "Document with error date format",
			args: args{
				docID: "test1",
				doc: map[string]interface{}{
					"name":       "Hello",
					"@timestamp": "2020-01-01T00:00:00Z8",
				},
				update: true,
			},
			wantErr: true,
		},
	}

	indexName := "TestDocument.index_1"
	var index *Index
	var err error
	t.Run("prepare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		err = StoreIndex(index)
		assert.NoError(t, err)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := index.CreateDocument(tt.args.docID, tt.args.doc, tt.args.update)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			// wait for WAL write to index
			time.Sleep(time.Second)

			assert.NoError(t, err)
			query := &meta.ZincQuery{
				Query: &meta.Query{
					Match: map[string]*meta.MatchQuery{
						"_all": {
							Query: "Hello",
						},
					},
				},
			}
			res, err := index.Search(query)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, res.Hits.Total.Value, 1)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err = DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}

func TestIndex_UpdateDocument(t *testing.T) {
	type args struct {
		docID  string
		doc    map[string]interface{}
		insert bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "update",
			args: args{
				docID: "1",
				doc: map[string]interface{}{
					"name": "HelloUpdate",
					"time": float64(1579098983),
				},
				insert: false,
			},
			wantErr: false,
		},
		{
			name: "Insert",
			args: args{
				docID: "2",
				doc: map[string]interface{}{
					"name": "HelloUpdate",
					"time": float64(1579098983),
				},
				insert: true,
			},
			wantErr: false,
		},
	}

	var index *Index
	var err error
	t.Run("prepare", func(t *testing.T) {
		index, err = NewIndex("TestIndex_UpdateDocument.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		err = StoreIndex(index)
		assert.NoError(t, err)
		prop := meta.NewProperty("date")
		mappings := index.GetMappings()
		mappings.SetProperty("time", prop)
		err = index.SetMappings(mappings)
		assert.NoError(t, err)

		err = index.CreateDocument("1", map[string]interface{}{
			"name": "Hello",
			"time": float64(1579098983),
		}, false)
		assert.NoError(t, err)

		// wait for WAL write to index
		time.Sleep(time.Second)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := index.UpdateDocument(tt.args.docID, tt.args.doc, tt.args.insert); (err != nil) != tt.wantErr {
				t.Errorf("Index.UpdateDocument() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err = DeleteIndex("TestIndex_UpdateDocument.index_1")
		assert.NoError(t, err)
	})
}

func TestIndex_DeleteDocument(t *testing.T) {
	type args struct {
		docID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				docID: "1",
			},
		},
		{
			name: "normal",
			args: args{
				docID: "2",
			},
			wantErr: true,
		},
	}
	indexName := "TestIndex_DeleteDocument.index_1"
	var index *Index
	var err error
	t.Run("prepare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		err = StoreIndex(index)
		assert.NoError(t, err)

		err = index.CreateDocument("1", map[string]interface{}{
			"name": "Hello",
			"time": float64(1579098983),
		}, false)
		assert.NoError(t, err)

		// wait for WAL write to index
		time.Sleep(time.Second)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := index.DeleteDocument(tt.args.docID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err = DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}

func TestDateLayoutDetection(t *testing.T) {
	type args struct {
		layout string
		input  string
	}
	tests := []args{
		{
			layout: "2006-01-02 15:04:05",
			input:  "2009-11-10 23:00:00",
		},
		{
			layout: "2006-01-02T15:04:05",
			input:  "2009-11-10T23:00:00",
		},
		{
			layout: time.RFC3339,
			input:  "2022-06-28T13:27:30+02:00",
		},
		{
			layout: "2006-01-02T15:04:05.999Z07:00",
			input:  "2022-06-28T13:27:30.789+02:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.layout, func(t *testing.T) {
			out := detectTimeLayout(tt.input)
			assert.Equal(t, out, tt.layout)
		})
	}
}

func TestIndex_CreateUpdateDocumentWithDateField(t *testing.T) {
	type fields struct {
		Name     string
		EpochMax int
		EpochMin int
	}
	type args struct {
		docID  string
		doc    map[string]interface{}
		update bool
	}
	tests := []struct {
		name         string
		fields       fields
		isRange      bool
		withResult   bool
		args         args
		wantQueryErr bool
	}{
		{
			name: "Document with invalid date type as string",
			fields: fields{
				Name: "updated_at",
			},
			withResult:   false,
			wantQueryErr: true,
			args: args{
				docID: "test_bad",
				doc: map[string]interface{}{
					"updated_at": "20091110 23:00:00",
				},
				update: false,
			},
		},
		{
			name: "Document with date type as string",
			fields: fields{
				Name:     "created_at",
				EpochMax: 1655972500041,
				EpochMin: 1243807200000,
			},
			withResult: true,
			args: args{
				docID: "test",
				doc: map[string]interface{}{
					"created_at": "2009-11-10T23:00:00",
				},
				update: false,
			},
		},
		{
			name: "Document with simple date type as string",
			fields: fields{
				Name:     "created_at_2",
				EpochMax: 1655972500041,
				EpochMin: 1243807200000,
			},
			withResult: true,
			args: args{
				docID: "test",
				doc: map[string]interface{}{
					"created_at_2": "2009-11-10 23:00:00",
				},
				update: true,
			},
		},
		{
			name: "Document with date array type as string",
			fields: fields{
				Name:     "time_range",
				EpochMax: 1669849200000,
				EpochMin: 1243807200000,
			},
			withResult: true,
			isRange:    true,
			args: args{
				docID: "test",
				doc: map[string]interface{}{
					"time_range": []interface{}{
						"2009-11-10T23:00:00",
						"2022-06-28T10:57:00",
					},
				},
				update: true,
			},
		},
		{
			name: "Document with simple date array type as string",
			fields: fields{
				Name:     "time_range_2",
				EpochMax: 1669849200000,
				EpochMin: 1243807200000,
			},
			withResult: true,
			isRange:    true,
			args: args{
				docID: "test",
				doc: map[string]interface{}{
					"time_range_2": []interface{}{
						"2009-11-10 23:00:00",
						"2022-06-28 10:57:00",
					},
				},
				update: true,
			},
		},
	}

	indexName := "TestDocument.WithDateField.index_1"
	var index *Index
	var err error
	t.Run("prepare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		err = StoreIndex(index)
		assert.NoError(t, err)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := index.CreateDocument(tt.args.docID, tt.args.doc, tt.args.update)
			assert.NoError(t, err)

			// wait for WAL write to index
			time.Sleep(time.Second)

			var query *meta.ZincQuery
			if tt.isRange {
				query = &meta.ZincQuery{
					Aggregations: map[string]meta.Aggregations{
						"agg_res": {
							DateHistogram: &meta.AggregationDateHistogram{
								Field:         tt.fields.Name,
								Format:        "epoch_millis",
								FixedInterval: "1d",
								ExtendedBounds: &aggregation.HistogramBound{
									Min: float64(tt.fields.EpochMin),
									Max: float64(tt.fields.EpochMax),
								},
							},
						},
					},
				}
			} else {
				query = &meta.ZincQuery{
					Query: &meta.Query{
						Range: map[string]*meta.RangeQuery{
							tt.fields.Name: {
								Format: "epoch_millis",
								GTE:    tt.fields.EpochMin,
							},
						},
					},
					Aggregations: map[string]meta.Aggregations{
						"agg_res": {
							DateHistogram: &meta.AggregationDateHistogram{
								Field:         tt.fields.Name,
								Format:        "epoch_millis",
								FixedInterval: "1d",
								ExtendedBounds: &aggregation.HistogramBound{
									Min: float64(tt.fields.EpochMin),
									Max: float64(tt.fields.EpochMax),
								},
							},
						},
					},
				}
			}

			res, err := index.Search(query)
			if tt.wantQueryErr {
				assert.Error(t, err)
				return

			}
			assert.NoError(t, err)

			no := 0
			if tt.withResult {
				no = 1
				if tt.isRange {
					no = 2
				}
			}

			assert.Equal(t, no, res.Hits.Total.Value)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err = DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}
