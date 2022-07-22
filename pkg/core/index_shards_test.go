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
)

func TestIndex_Shards(t *testing.T) {
	type args struct {
		docID string
		doc   map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "document",
			args: args{
				docID: "",
				doc: map[string]interface{}{
					"name": "Hello1",
					"time": float64(time.Now().UnixNano()),
				},
			},
			wantErr: false,
		},
		{
			name: "document",
			args: args{
				docID: "",
				doc: map[string]interface{}{
					"name": "Hello2",
					"time": float64(time.Now().UnixNano()),
				},
			},
			wantErr: false,
		},
	}

	var index *Index
	var err error
	var indexName = "TestIndex_Shards.index_1"
	t.Run("perpare", func(t *testing.T) {
		index, err = NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = StoreIndex(index)
		assert.NoError(t, err)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := index.CreateDocument(tt.args.docID, tt.args.doc, false)
			assert.NoError(t, err)

			// wait for WAL write to index
			time.Sleep(time.Second)

			if err := index.GetShardByDocID(tt.args.docID).NewShard(); (err != nil) != tt.wantErr {
				t.Errorf("Index.NewShard() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := index.CheckShards(); (err != nil) != tt.wantErr {
				t.Errorf("Index.CheckShards() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err := DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}
