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
	"sync"
	"sync/atomic"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/analysis"
	"github.com/goccy/go-json"
	"golang.org/x/sync/errgroup"

	"github.com/zinclabs/zinc/pkg/meta"
	zincanalysis "github.com/zinclabs/zinc/pkg/uquery/analysis"
)

type Index struct {
	ref          *meta.Index
	analyzers    map[string]*analysis.Analyzer
	shards       []*IndexShard
	shardNum     int64
	shardNumUint uint64 // just for do HASH
	lock         sync.RWMutex
}

func (index *Index) MarshalJSON() ([]byte, error) {
	index.lock.RLock()
	b, err := json.Marshal(index.ref)
	index.lock.RUnlock()
	return b, err
}

func (index *Index) GetIndex() meta.Index {
	return *index.ref
}

func (index *Index) GetShardNum() int64 {
	return index.shardNum
}

func (index *Index) GetName() string {
	return index.ref.Name
}

func (index *Index) GetStorageType() string {
	return index.ref.StorageType
}

func (index *Index) GetMappings() *meta.Mappings {
	index.lock.RLock()
	m := index.ref.Mappings
	index.lock.RUnlock()
	return m
}

func (index *Index) GetSettings() *meta.IndexSettings {
	index.lock.RLock()
	s := index.ref.Settings
	index.lock.RUnlock()
	return s
}

func (index *Index) GetStats() meta.IndexStat {
	index.lock.RLock()
	s := index.ref.Stats
	index.lock.RUnlock()
	return s
}

func (index *Index) GetAnalyzers() map[string]*analysis.Analyzer {
	index.lock.RLock()
	a := index.analyzers
	index.lock.RUnlock()
	return a
}

func (index *Index) UseTemplate() error {
	template, err := UseTemplate(index.GetName())
	if err != nil {
		return err
	}

	if template == nil {
		return nil
	}

	if template.Template.Settings != nil {
		// update settings
		_ = index.SetSettings(template.Template.Settings)
		// update analyzers
		analyzers, _ := zincanalysis.RequestAnalyzer(template.Template.Settings.Analysis)
		_ = index.SetAnalyzers(analyzers)
	}

	if template.Template.Mappings != nil {
		_ = index.SetMappings(template.Template.Mappings)
	}

	return nil
}

func (index *Index) SetSettings(settings *meta.IndexSettings) error {
	if settings == nil {
		return nil
	}

	index.lock.Lock()
	index.ref.Settings = settings
	index.lock.Unlock()

	return nil
}

func (index *Index) SetAnalyzers(analyzers map[string]*analysis.Analyzer) error {
	if len(analyzers) == 0 {
		return nil
	}

	index.lock.Lock()
	index.analyzers = analyzers
	index.lock.Unlock()

	return nil
}

func (index *Index) SetMappings(mappings *meta.Mappings) error {
	if mappings == nil {
		mappings = meta.NewMappings()
	}

	// custom analyzer just for text field
	for field, prop := range mappings.ListProperty() {
		if prop.Type != "text" {
			prop.Analyzer = ""
			prop.SearchAnalyzer = ""
			mappings.SetProperty(field, prop)
		}
	}

	mappings.SetProperty("_id", meta.NewProperty("keyword"))

	// @timestamp need date_range/date_histogram aggregation, and mappings used for type check in aggregation
	fieldTimestamp, exists := mappings.GetProperty(meta.TimeFieldName)
	if !exists {
		fieldTimestamp = meta.NewProperty("date")
	}
	fieldTimestamp.Index = true
	fieldTimestamp.Sortable = true
	fieldTimestamp.Aggregatable = true
	mappings.SetProperty(meta.TimeFieldName, fieldTimestamp)

	// update in the cache
	index.lock.Lock()
	index.ref.Mappings = mappings
	index.lock.Unlock()

	return nil
}

func (index *Index) GetWALSize() uint64 {
	size := uint64(0)
	for _, shard := range index.shards {
		s, err := shard.GetWALSize()
		if err != nil {
			return size
		}
		size += s
	}
	return size
}

// GetReaders return all shard readers
func (index *Index) GetReaders(timeMin, timeMax int64) ([]*bluge.Reader, error) {
	readers := make([]*bluge.Reader, 0)
	for _, shard := range index.shards {
		rs, err := shard.GetReaders(timeMin, timeMax)
		if err != nil {
			return nil, err
		}
		if len(rs) > 0 {
			readers = append(readers, rs...)
		}
	}
	return readers, nil
}

func (index *Index) UpdateMetadata() error {
	var totalDocNum, totalSize uint64
	for i := int64(0); i < index.shardNum; i++ {
		totalDocNum += atomic.LoadUint64(&index.shards[i].ref.Stats.DocNum)
		totalSize += atomic.LoadUint64(&index.shards[i].ref.Stats.StorageSize)
	}

	if totalDocNum > 0 && totalSize > 0 {
		index.lock.Lock()
		atomic.StoreUint64(&index.ref.Stats.DocNum, totalDocNum)
		atomic.StoreUint64(&index.ref.Stats.StorageSize, totalSize)
		index.lock.Unlock()
	}

	return storeIndex(index)
}

func (index *Index) UpdateMetadataByShard(n int64) {
	var totalDocNum, totalSize uint64
	// update docNum and storageSize
	shard := index.shards[n]
	for i := int64(0); i < shard.GetShardNum(); i++ {
		index.UpdateStatsBySecondShard(n, i)
		totalDocNum += atomic.LoadUint64(&shard.ref.Shards[i].Stats.DocNum)
		totalSize += atomic.LoadUint64(&shard.ref.Shards[i].Stats.StorageSize)
	}
	if totalDocNum > 0 && totalSize > 0 {
		index.lock.Lock()
		atomic.StoreUint64(&shard.ref.Stats.DocNum, totalDocNum)
		atomic.StoreUint64(&shard.ref.Stats.StorageSize, totalSize)
		index.lock.Unlock()
	}

	// update latest shard docTime
	secondShard := index.shards[shard.GetLatestShardID()]
	index.lock.Lock()
	atomic.StoreInt64(&secondShard.ref.Stats.DocTimeMin, atomic.LoadInt64(&shard.ref.Stats.DocTimeMin))
	atomic.StoreInt64(&secondShard.ref.Stats.DocTimeMax, atomic.LoadInt64(&shard.ref.Stats.DocTimeMax))
	index.lock.Unlock()
}

func (index *Index) UpdateStatsBySecondShard(n, secondIndex int64) {
	shard := index.shards[n]
	shard.lock.RLock()
	secondShard := shard.shards[secondIndex]
	shard.lock.RUnlock()

	secondShard.lock.RLock()
	w := secondShard.writer
	secondShard.lock.RUnlock()
	if w == nil {
		return
	}

	var docNum, storageSize uint64
	_, storageSize = w.DirectoryStats()
	if r, err := w.Reader(); err == nil {
		if n, err := r.Count(); err == nil {
			docNum = n
		}
		_ = r.Close()
	}

	index.lock.Lock()
	if docNum > 0 {
		atomic.StoreUint64(&secondShard.ref.Stats.DocNum, docNum)
	}
	if storageSize > 0 {
		atomic.StoreUint64(&secondShard.ref.Stats.StorageSize, storageSize)
	}
	index.lock.Unlock()
}

// Reopen just close the index, it will open automatically by trigger
// Deprecated: it will be removed in the future
func (index *Index) Reopen() error {
	return index.Close()
}

func (index *Index) Close() error {
	if err := index.UpdateMetadata(); err != nil {
		return err
	}

	eg := errgroup.Group{}
	for _, shard := range index.shards {
		shard := shard
		eg.Go(func() error {
			return shard.Close()
		})
	}
	return eg.Wait()
}
