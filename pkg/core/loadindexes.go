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
	"github.com/rs/zerolog/log"

	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/metadata"
	zincanalysis "github.com/zinclabs/zinc/pkg/uquery/analysis"
)

func LoadZincIndexesFromMetadata() error {
	indexes, err := metadata.Index.List(0, 0)
	if err != nil {
		return err
	}

	for i := range indexes {
		readIndex := indexes[i]
		index := new(Index)
		index.ref = new(meta.Index)
		index.ref.Name = readIndex.Name
		index.ref.StorageType = readIndex.StorageType
		index.ref.Settings = readIndex.Settings
		index.ref.Mappings = readIndex.Mappings
		index.ref.Stats = readIndex.Stats

		index.ref.ShardNum = readIndex.ShardNum
		index.ref.Shards = make([]*meta.IndexShard, index.ref.ShardNum)
		for i := range readIndex.Shards {
			index.ref.Shards[i] = &meta.IndexShard{
				ID:       readIndex.Shards[i].ID,
				ShardNum: readIndex.Shards[i].ShardNum,
				Stats:    readIndex.Shards[i].Stats,
			}
			index.ref.Shards[i].Shards = make([]*meta.IndexSecondShard, index.ref.Shards[i].ShardNum)
			for j := range readIndex.Shards[i].Shards {
				index.ref.Shards[i].Shards[j] = &meta.IndexSecondShard{
					ID:    readIndex.Shards[i].Shards[j].ID,
					Stats: readIndex.Shards[i].Shards[j].Stats,
				}
			}
		}

		// init shards wrapper
		index.shardNum = index.ref.ShardNum
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

		log.Info().Msgf("Loading  index... [%s:%s] shards[%d]", index.ref.Name, index.ref.StorageType, index.ref.ShardNum)

		// upgrade from version <= 0.2.4
		// TODO v0.2.4 -> v0.2.7
		// TODO v0.2.5 -> v0.2.7
		// TODO v0.2.6 -> v0.2.7

		// load index analysis
		if index.ref.Settings != nil && index.ref.Settings.Analysis != nil {
			index.analyzers, err = zincanalysis.RequestAnalyzer(index.ref.Settings.Analysis)
			if err != nil {
				return errors.New(errors.ErrorTypeRuntimeException, "parse stored analysis error").Cause(err)
			}
		}

		// load in memory
		ZINC_INDEX_LIST.Add(index)
	}

	return nil
}
