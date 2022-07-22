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
	"fmt"
	"regexp"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/analysis"

	"github.com/zinclabs/zinc/pkg/bluge/directory"
	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/metadata"
)

var indexNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func CheckIndexName(name string) error {
	if name == "" {
		return fmt.Errorf("index name cannot be empty")
	}
	if strings.HasPrefix(name, "_") {
		return fmt.Errorf("index name cannot start with _")
	}
	if !indexNameRe.Match([]byte(name)) {
		return fmt.Errorf("index name [%s] is invalid, just accept [a-zA-Z0-9_.-]", name)
	}
	return nil
}

// NewIndex creates an instance of a physical zinc index that can be used to store and retrieve data.
func NewIndex(name, storageType string, shardNum int64) (*Index, error) {
	if err := CheckIndexName(name); err != nil {
		return nil, err
	}

	if storageType == "" {
		storageType = "disk"
	}

	if shardNum <= 0 {
		shardNum = config.Global.Shard.Num
	}

	index := new(Index)
	index.ref = new(meta.Index)
	index.ref.Name = name
	index.ref.StorageType = storageType

	// use template
	if err := index.UseTemplate(); err != nil {
		return nil, err
	}
	if index.ref.Settings != nil {
		if index.ref.Settings.NumberOfShards == 0 {
			index.ref.Settings.NumberOfShards = shardNum
		} else {
			shardNum = index.ref.Settings.NumberOfShards
		}
	}

	index.ref.ShardNum = shardNum
	index.shardNum = shardNum
	for i := int64(0); i < index.shardNum; i++ {
		shard := &meta.IndexShard{ID: i, ShardNum: 1}
		for j := int64(0); j < shard.ShardNum; j++ {
			shard.Shards = append(shard.Shards, &meta.IndexSecondShard{ID: j})
		}
		index.ref.Shards = append(index.ref.Shards, shard)
	}

	// init shards wrapper
	index.shardNumUint = uint64(index.shardNum)
	index.shards = make([]*IndexShard, index.shardNum)
	for i := range index.ref.Shards {
		index.shards[i] = &IndexShard{root: index, ref: index.ref.Shards[i]}
		index.shards[i].shards = make([]*IndexSecondShard, index.ref.Shards[i].ShardNum)
		for j := range index.ref.Shards[i].Shards {
			index.shards[i].shards[j] = &IndexSecondShard{
				root: index,
				ref:  index.ref.Shards[i].Shards[j],
			}
		}
	}

	return index, nil
}

// LoadIndexWriter load the index writer from the storage
func OpenIndexWriter(name string, storageType string, defaultSearchAnalyzer *analysis.Analyzer, timeRange ...int64) (*bluge.Writer, error) {
	cfg := getOpenConfig(name, storageType, defaultSearchAnalyzer, timeRange...)
	return bluge.OpenWriter(cfg)
}

func getOpenConfig(name string, storageType string, defaultSearchAnalyzer *analysis.Analyzer, timeRange ...int64) bluge.Config {
	var dataPath string
	var cfg bluge.Config
	switch storageType {
	case "s3":
		dataPath = config.Global.S3.Bucket
		cfg = directory.GetS3Config(dataPath, name, timeRange...)
	case "minio":
		dataPath = config.Global.MinIO.Bucket
		cfg = directory.GetMinIOConfig(dataPath, name, timeRange...)
	default:
		dataPath = config.Global.DataPath
		cfg = directory.GetDiskConfig(dataPath, name, timeRange...)
	}
	if defaultSearchAnalyzer != nil {
		cfg.DefaultSearchAnalyzer = defaultSearchAnalyzer
	}
	return cfg
}

// storeIndex stores the index to metadata
func StoreIndex(index *Index) error {
	// check index
	checkIndex(index)
	// store index
	if err := storeIndex(index); err != nil {
		return err
	}
	// cache index
	ZINC_INDEX_LIST.Add(index)
	return nil
}

func checkIndex(index *Index) {
	index.lock.Lock()

	if index.ref.Settings == nil {
		index.ref.Settings = new(meta.IndexSettings)
	}
	if index.ref.Settings != nil && index.ref.Settings.NumberOfShards == 0 {
		index.ref.Settings.NumberOfShards = index.ref.ShardNum
	}
	if index.ref.Mappings == nil {
		// set default mappings
		index.ref.Mappings = meta.NewMappings()
		index.ref.Mappings.SetProperty(meta.TimeFieldName, meta.NewProperty("date"))
	}
	if index.analyzers == nil {
		index.analyzers = make(map[string]*analysis.Analyzer)
	}

	index.lock.Unlock()
}

func storeIndex(index *Index) error {
	data, err := index.MarshalJSON()
	if err != nil {
		return fmt.Errorf("core.storeIndex: index: %s, error: %s", index.ref.Name, err.Error())
	}
	err = metadata.Index.Set(index.GetName(), data)
	if err != nil {
		return fmt.Errorf("core.storeIndex: index: %s, error: %s", index.ref.Name, err.Error())
	}
	return nil
}

func GetIndex(name string) (*Index, bool) {
	return ZINC_INDEX_LIST.Get(name)
}

func GetOrCreateIndex(name, storageType string, shardNum int64) (*Index, bool, error) {
	return ZINC_INDEX_LIST.GetOrCreate(name, storageType, shardNum)
}
