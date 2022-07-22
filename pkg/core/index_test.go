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
	"reflect"
	"testing"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/analysis"
	"github.com/blugelabs/bluge/analysis/analyzer"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/meta"
)

func TestIndex_Index(t *testing.T) {
	indexName := "TestIndex_Index.index_1"
	index, err := NewIndex(indexName, "disk", 2)
	assert.NoError(t, err)
	assert.NotNil(t, index)

	err = StoreIndex(index)
	assert.NoError(t, err)

	got, err := json.Marshal(index)
	assert.NoError(t, err)
	assert.NotNil(t, got)

	name := index.GetName()
	assert.Equal(t, indexName, name)

	settings := index.GetSettings()
	assert.NotNil(t, settings)

	mappings := index.GetMappings()
	assert.NotNil(t, mappings)

	analyzers := index.GetAnalyzers()
	assert.NotNil(t, analyzers)

	err = index.Reopen()
	assert.NoError(t, err)
}

func TestIndex_BuildBlugeDocumentFromJSON(t *testing.T) {
	var index *Index
	var err error
	indexName := "TestIndex_BuildBlugeDocumentFromJSON.index_1"

	type args struct {
		docID string
		doc   map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		init    func()
		want    *bluge.Document
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				docID: "1",
				doc: map[string]interface{}{
					"id":     "1",
					"name":   "test1",
					"age":    10,
					"length": 3.14,
					"dev":    true,
					"address": map[string]interface{}{
						"street": "447 Great Mall Dr",
						"city":   "Milpitas",
						"state":  "CA",
						"zip":    95035,
					},
					"tag1":       []interface{}{"tag1", "tag2"},
					"tag2":       []interface{}{3.14, 3.15},
					"tag3":       []interface{}{true, false},
					"@timestamp": time.Now().Format(time.RFC3339),
					"time":       time.Now().Format(time.RFC3339),
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "timestamp with epoch_millis",
			args: args{
				docID: "2",
				doc: map[string]interface{}{
					"id":         "2",
					"name":       "test1",
					"age":        10,
					"length":     3.14,
					"dev":        true,
					"@timestamp": float64(1652176732575),
					"time":       float64(1652176732575),
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "timestamp with format",
			args: args{
				docID: "2",
				doc: map[string]interface{}{
					"id":         "2",
					"name":       "test1",
					"age":        10,
					"length":     3.14,
					"dev":        true,
					"@timestamp": time.Now().Format("2006-01-02 15:04:05.000"),
					"time":       time.Now().Format("2006-01-02 15:04:05.000"),
				},
			},
			init: func() {
				index.GetMappings().SetProperty("@timestamp", meta.Property{
					Type:   "time",
					Index:  true,
					Format: "2006-01-02 15:04:05.000",
				})
				index.GetMappings().SetProperty("time", meta.Property{
					Type:   "time",
					Index:  true,
					Format: "2006-01-02 15:04:05.000",
				})
			},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "with analyzer",
			args: args{
				docID: "3",
				doc: map[string]interface{}{
					"id":     "3",
					"name":   "test",
					"age":    "10",
					"length": 3,
					"dev":    true,
				},
			},
			init: func() {
				index.GetMappings().SetProperty("id", meta.Property{
					Type:         "keyword",
					Index:        true,
					Aggregatable: true,
				})
				index.GetMappings().SetProperty("name", meta.Property{
					Type:          "text",
					Analyzer:      "analyzer_1",
					Index:         true,
					Highlightable: true,
				})
				index.analyzers["analyzer_1"] = analyzer.NewStandardAnalyzer()
			},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "type conflict text",
			args: args{
				docID: "4",
				doc: map[string]interface{}{
					"id":   "4",
					"name": 3,
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "type conflict numeric",
			args: args{
				docID: "4",
				doc: map[string]interface{}{
					"id":     "4",
					"name":   "test1",
					"age":    "10",
					"length": 3,
					"dev":    true,
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "keyword type float64",
			args: args{
				docID: "4",
				doc: map[string]interface{}{
					"id": 3.14,
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "keyword type int",
			args: args{
				docID: "4",
				doc: map[string]interface{}{
					"id": 3,
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "keyword type bool",
			args: args{
				docID: "4",
				doc: map[string]interface{}{
					"id": false,
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
		{
			name: "keyword type other",
			args: args{
				docID: "4",
				doc: map[string]interface{}{
					"id": []byte("foo"),
				},
			},
			init:    func() {},
			want:    &bluge.Document{},
			wantErr: false,
		},
	}

	t.Run("prepare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = StoreIndex(index)
		assert.NoError(t, err)
		index.GetMappings().SetProperty("time", meta.NewProperty("date"))
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.init()
			got, err := index.CheckDocument(tt.args.docID, tt.args.doc, false, 0)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, got)
			var doc map[string]interface{}
			err = json.Unmarshal(got, &doc)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			got2, err := index.BuildBlugeDocumentFromJSON(tt.args.docID, doc)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, got2)
			wantType := reflect.TypeOf(tt.want)
			gotType := reflect.TypeOf(got2)
			assert.Equal(t, wantType, gotType)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err := DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}

func TestIndex_Settings(t *testing.T) {
	var index *Index
	var err error
	indexName := "TestIndex_Settings.index_1"

	t.Run("prepare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("setting", func(t *testing.T) {
		err := index.SetSettings(&meta.IndexSettings{
			NumberOfShards:   1,
			NumberOfReplicas: 0,
			Analysis: &meta.IndexAnalysis{
				Analyzer: map[string]*meta.Analyzer{
					"default": {
						Type: "standard",
					},
				},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("mapping", func(t *testing.T) {
		err := index.SetMappings(&meta.Mappings{
			Properties: map[string]meta.Property{
				"id": meta.NewProperty("keyword"),
			},
		})
		assert.NoError(t, err)
	})

	t.Run("analyzer", func(t *testing.T) {
		err := index.SetAnalyzers(map[string]*analysis.Analyzer{
			"standard": analyzer.NewStandardAnalyzer(),
		})
		assert.NoError(t, err)
	})

	t.Run("cleanup", func(t *testing.T) {
		err := DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}
