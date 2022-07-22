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
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/zinclabs/zinc/pkg/config"
)

// Record opened WAL used to do consume
var ZINC_INDEX_SHARD_WAL_LIST IndexShardWALList

type IndexShardWALList struct {
	Shards map[string]*IndexShard
	lock   sync.RWMutex
}

func init() {
	ZINC_INDEX_SHARD_WAL_LIST.Shards = make(map[string]*IndexShard)
	go ZINC_INDEX_SHARD_WAL_LIST.ConsumeWAL()
}

func (t *IndexShardWALList) Add(shard *IndexShard) {
	t.lock.Lock()
	t.Shards[shard.GetShardName()] = shard
	t.lock.Unlock()
}

func (t *IndexShardWALList) Remove(name string) {
	t.lock.Lock()
	delete(t.Shards, name)
	t.lock.Unlock()
}

func (t *IndexShardWALList) List() []*IndexShard {
	t.lock.RLock()
	shards := make([]*IndexShard, 0, len(t.Shards))
	for _, shard := range t.Shards {
		shards = append(shards, shard)
	}
	t.lock.RUnlock()
	return shards
}

func (t *IndexShardWALList) Len() int {
	t.lock.RLock()
	n := len(t.Shards)
	t.lock.RUnlock()
	return n
}

func (t *IndexShardWALList) ConsumeWAL() {
	interval, err := parseInterval(config.Global.WalSyncInterval)
	if err != nil {
		log.Fatal().Err(err).Msg("consume ParseInterval")
	}
	tick := time.NewTicker(interval)
	for range tick.C {
		eg := &errgroup.Group{}
		eg.SetLimit(config.Global.ReadGorutineNum)
		indexClosed := make(chan string, t.Len())
		for _, shard := range t.List() {
			shard := shard
			eg.Go(func() error {
				select {
				case <-shard.close:
					indexClosed <- shard.GetShardName()
					return nil
				default:
					// continue
				}
				shard.ConsumeWAL()
				return nil
			})
		}
		_ = eg.Wait()
		close(indexClosed)

		// check index closed
		for name := range indexClosed {
			t.Remove(name)
		}
	}
}
