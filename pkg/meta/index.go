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

package meta

type Index struct {
	Name        string         `json:"name"`
	StorageType string         `json:"storage_type"`
	Settings    *IndexSettings `json:"settings,omitempty"`
	Mappings    *Mappings      `json:"mappings,omitempty"`
	ShardNum    int64          `json:"shard_num"`
	Shards      []*IndexShard  `json:"shards"`
	Stats       IndexStat      `json:"stats"`
}

type IndexShard struct {
	ID       int64               `json:"id"`
	NodeID   string              `json:"node_id"` // remote instance ID
	ShardNum int64               `json:"shard_num"`
	Shards   []*IndexSecondShard `json:"shards"`
	Stats    IndexStat           `json:"stats"`
}

type IndexSecondShard struct {
	ID    int64     `json:"id"`
	Stats IndexStat `json:"stats"`
}

type IndexStat struct {
	DocNum      uint64 `json:"doc_num"`
	DocTimeMin  int64  `json:"doc_time_min"`
	DocTimeMax  int64  `json:"doc_time_max"`
	StorageSize uint64 `json:"storage_size"`
	WALSize     uint64 `json:"wal_size"`
}

type IndexSimple struct {
	Name        string                 `json:"name"`
	StorageType string                 `json:"storage_type"`
	ShardNum    int64                  `json:"shard_num"`
	Settings    *IndexSettings         `json:"settings,omitempty"`
	Mappings    map[string]interface{} `json:"mappings,omitempty"`
}

type IndexSettings struct {
	NumberOfShards   int64          `json:"-"`
	NumberOfReplicas int64          `json:"-"`
	Analysis         *IndexAnalysis `json:"analysis,omitempty"`
}

type IndexAnalysis struct {
	Analyzer    map[string]*Analyzer   `json:"analyzer,omitempty"`
	CharFilter  map[string]interface{} `json:"char_filter,omitempty"`
	Tokenizer   map[string]interface{} `json:"tokenizer,omitempty"`
	TokenFilter map[string]interface{} `json:"token_filter,omitempty"`
	Filter      map[string]interface{} `json:"filter,omitempty"` // compatibility with es, alias for TokenFilter
}
