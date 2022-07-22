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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/blugelabs/bluge"
	blugeindex "github.com/blugelabs/bluge/index"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/wal"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// OpenWAL open WAL for index
func (s *IndexShard) OpenWAL() error {
	if atomic.LoadUint32(&s.open) == 1 {
		return nil
	}

	// open wal
	s.lock.Lock()
	// enter here check wal isopen again
	if s.wal != nil {
		s.lock.Unlock()
		return nil
	}
	// do open wal
	var err error
	if s.wal, err = wal.Open(s.GetShardName()); err != nil {
		s.lock.Unlock()
		return err
	}
	s.lock.Unlock()

	// check wal rollback
	if err = s.Rollback(); err != nil {
		return err
	}

	// set wal opened
	atomic.StoreUint32(&s.open, 1)
	s.close = make(chan struct{})

	// set wal to consumer list
	ZINC_INDEX_SHARD_WAL_LIST.Add(s)

	return nil
}

func (s *IndexShard) Rollback() error {
	readMinID, readMaxID, err := s.readRedoLog(RedoActionRead)
	// fmt.Println("readMinID:", readMinID, "readMaxID:", readMaxID)
	if err != nil {
		// key not exists, no need to rollback
		if err.Error() == errors.ErrNotFound.Error() {
			return nil
		}
		return err
	}
	writeMinID, writeMaxID, err := s.readRedoLog(RedoActionWrite)
	// fmt.Println("writeMinID:", writeMinID, "writeMaxID:", writeMaxID)
	if err != nil {
		// key not exists, need to rollback
		if err.Error() != errors.ErrNotFound.Error() {
			return err
		}
	}
	if readMinID == writeMinID && readMaxID == writeMaxID {
		// it's ok, no need to rollback
		return nil
	}

	// Rollback
	log.Info().
		Str("index", s.GetIndexName()).
		Int64("shard", s.GetID()).
		Uint64("minID", readMinID).
		Uint64("maxID", readMaxID).
		Msg("rollback start")

	var entry []byte
	batch := blugeindex.NewBatch()
	docs := make(walMergeDocs)
	for minID := readMinID; minID <= readMaxID; minID++ {
		entry, err = s.wal.Read(minID)
		if err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("rollback wal.Read()")
			return err
		}

		doc := make(map[string]interface{})
		err = json.Unmarshal(entry, &doc)
		if err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("rollback wal.entry.Unmarshal()")
			return err
		}
		docs.AddDocument(doc)
	}

	err = docs.WriteTo(s, batch, true)
	if err != nil {
		return err
	}

	// Truncate log
	minID := readMinID - 1 // minID should be last successfully committed ID
	if err = s.wal.TruncateFront(minID); err != nil {
		log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Uint64("id", minID).Msg("rollback wal.Truncate()")
	}

	log.Info().Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Uint64("minID", readMinID).Uint64("maxID", readMaxID).Msg("rollback success")
	return nil
}

func (s *IndexShard) ConsumeWAL() {
	select {
	case <-s.close:
		return
	default:
		// continue
	}

	if err := s.wal.Sync(); err != nil {
		log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.Sync()")
	}

	var err error
	var entry []byte
	var minID, maxID, startID uint64
	maxID, err = s.wal.LastIndex()
	if err != nil {
		log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.LastIndex()")
		return
	}
	// read last committed ID
	_, minID, err = s.readRedoLog(RedoActionWrite)
	if err != nil && err.Error() != errors.ErrNotFound.Error() {
		log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.readRedoLog()")
		return
	}
	if minID == maxID {
		return // no new entries
	}
	// log.Debug().Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Uint64("minID", minID).Uint64("maxID", maxID).Msg("consume wal begin")

	batch := blugeindex.NewBatch()
	docs := make(walMergeDocs)
	minID++
	for startID = minID; minID <= maxID; minID++ {
		entry, err = s.wal.Read(minID)
		if err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.Read()")
			return
		}

		doc := make(map[string]interface{})
		err = json.Unmarshal(entry, &doc)
		if err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.entry.Unmarshal()")
			return
		}
		docs.AddDocument(doc)
		if docs.MaxShardLen() >= config.Global.BatchSize {
			if err = s.writeRedoLog(RedoActionRead, startID, minID); err != nil {
				log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Str("stage", "read").Msg("consume wal.redolog.Write()")
				return
			}
			if err = docs.WriteTo(s, batch, false); err != nil {
				log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.docs.WriteTo()")
				return
			}
			if err = s.writeRedoLog(RedoActionWrite, startID, minID); err != nil {
				log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Str("stage", "write").Msg("consume wal.redolog.Write()")
				return
			}
			// Reset startID to nextID
			startID = minID + 1
		}
	}

	minID-- // need reduce one, because the next loop add one

	// check if there is any docs to write
	if docs.MaxShardLen() > 0 {
		if err = s.writeRedoLog(RedoActionRead, startID, minID); err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Str("stage", "read").Msg("consume wal.redolog.Write()")
			return
		}
		if err := docs.WriteTo(s, batch, false); err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume wal.docs.WriteTo()")
			return
		}
		if err = s.writeRedoLog(RedoActionWrite, startID, minID); err != nil {
			log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Str("stage", "write").Msg("consume wal.redolog.Write()")
			return
		}
	}
	// log.Debug().Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Uint64("minID", minID).Uint64("maxID", maxID).Msg("consume wal end")

	// Truncate log
	if err = s.wal.TruncateFront(minID); err != nil {
		log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Uint64("id", minID).Msg("consume wal.Truncate()")
		return
	}

	// check shards
	if err = s.CheckShards(); err != nil {
		log.Error().Err(err).Str("index", s.GetIndexName()).Int64("shard", s.GetID()).Msg("consume index.CheckShards()")
		return
	}

	//  update metadata
	s.root.UpdateMetadataByShard(s.GetID())
}

const (
	RedoActionRead     = uint64(1)
	RedoActionWrite    = uint64(2)
	RedoActionTruncate = uint64(3)
)

func (s *IndexShard) writeRedoLog(option uint64, minID, maxID uint64) error {
	value := fmt.Sprintf("%d:%d", minID, maxID)
	return s.wal.Redo.Write(option, []byte(value))
}

func (s *IndexShard) readRedoLog(option uint64) (uint64, uint64, error) {
	v, err := s.wal.Redo.Read(option)
	if err != nil {
		return 0, 0, err
	}
	vs := strings.Split(string(v), ":")
	if len(vs) != 2 {
		return 0, 0, fmt.Errorf("invalid redo log: [%s]", string(v))
	}
	minID, err := zutils.ToUint64(vs[0])
	if err != nil {
		return 0, 0, err
	}
	maxID, err := zutils.ToUint64(vs[1])
	if err != nil {
		return 0, 0, err
	}
	return minID, maxID, nil
}

func (s *IndexShard) GetWALSize() (uint64, error) {
	if s.wal == nil {
		return 0, nil
	}
	return s.wal.Len()
}

type walDocument struct {
	docID   string
	actions []string
	data    map[string]interface{}
}

type walMergeDocs map[int64]map[string]*walDocument

func (w *walMergeDocs) MaxShardLen() int {
	n := 0
	for _, v := range *w {
		if len(v) > n {
			n = len(v)
		}
	}
	return n
}

func (w *walMergeDocs) Reset() {
	for _, v := range *w {
		for k := range v {
			delete(v, k)
		}
	}
}

func (w *walMergeDocs) AddDocument(data map[string]interface{}) {
	action := data[meta.ActionFieldName].(string)
	docID := data[meta.IDFieldName].(string)
	shardID := int64(data[meta.ShardFieldName].(float64))
	shard, ok := (*w)[shardID]
	if !ok {
		shard = make(map[string]*walDocument)
		(*w)[shardID] = shard
	}
	doc, ok := shard[docID]
	if !ok {
		doc = &walDocument{docID: docID}
		shard[docID] = doc
	}
	doc.actions = append(doc.actions, action)
	doc.data = data
}

// WriteTo write documents to index and sync to disk
// need split by shards
// need merge actions by docID
func (w *walMergeDocs) WriteTo(shard *IndexShard, batch *blugeindex.Batch, rollback bool) error {
	var err error
	for shardID := range *w {
		if !rollback {
			err = w.WriteToShard(shard, shardID, batch)
		} else {
			err = w.WriteToShardRollback(shard, shardID, batch)
		}
		if err != nil {
			return err
		}
		batch.Reset()
	}
	w.Reset()
	return nil
}

func (w *walMergeDocs) WriteToShard(shard *IndexShard, shardID int64, batch *blugeindex.Batch) error {
	docs, ok := (*w)[shardID]
	if !ok {
		return nil
	}
	var writer *bluge.Writer
	otherWriters := make([]*bluge.Writer, 0)
	otherBatch := blugeindex.NewBatch()
	if shardID >= 0 {
		w, err := shard.GetWriter(shardID)
		if err != nil {
			return err
		}
		writer = w
	} else {
		ws, err := shard.GetWriters() // get all shard
		if err != nil {
			return err
		}
		writer = ws[len(ws)-1]
		otherWriters = append(otherWriters, ws...)
		otherWriters = otherWriters[:len(ws)-1]
	}
	var firstAction, lastAction string
	for _, doc := range docs {
		// str, err := json.Marshal(doc.data)
		// fmt.Printf("%s, %v, %v\n", str, err, doc.actions)
		bdoc, err := shard.BuildBlugeDocumentFromJSON(doc.docID, doc.data)
		if err != nil {
			return err
		}
		firstAction = doc.actions[0]
		switch firstAction {
		case meta.ActionTypeInsert:
			if len(doc.actions) == 1 {
				batch.Insert(bdoc)
			} else {
				lastAction = doc.actions[len(doc.actions)-1]
				switch lastAction {
				case meta.ActionTypeInsert:
					batch.Insert(bdoc)
				case meta.ActionTypeUpdate:
					batch.Insert(bdoc)
				case meta.ActionTypeDelete:
					// noop
				}
			}
		case meta.ActionTypeUpdate:
			if len(doc.actions) == 1 {
				batch.Update(bdoc.ID(), bdoc)
				otherBatch.Delete(bdoc.ID())
			} else {
				lastAction = doc.actions[len(doc.actions)-1]
				switch lastAction {
				case meta.ActionTypeInsert:
					batch.Update(bdoc.ID(), bdoc)
					otherBatch.Delete(bdoc.ID())
				case meta.ActionTypeUpdate:
					batch.Update(bdoc.ID(), bdoc)
					otherBatch.Delete(bdoc.ID())
				case meta.ActionTypeDelete:
					batch.Delete(bdoc.ID())
					otherBatch.Delete(bdoc.ID())
				}
			}
		case meta.ActionTypeDelete:
			if len(doc.actions) == 1 {
				batch.Delete(bdoc.ID())
				otherBatch.Delete(bdoc.ID())
			} else {
				lastAction = doc.actions[len(doc.actions)-1]
				switch lastAction {
				case meta.ActionTypeInsert:
					batch.Update(bdoc.ID(), bdoc)
					otherBatch.Delete(bdoc.ID())
				case meta.ActionTypeUpdate:
					batch.Update(bdoc.ID(), bdoc)
					otherBatch.Delete(bdoc.ID())
				case meta.ActionTypeDelete:
					batch.Delete(bdoc.ID())
					otherBatch.Delete(bdoc.ID())
				}
			}
		default:
			return fmt.Errorf("walMergeDocs: invalid action type [%s]", firstAction)
		}
	}

	if err := writer.Batch(batch); err != nil {
		return err
	}
	for _, writer := range otherWriters {
		if err := writer.Batch(otherBatch); err != nil {
			return err
		}
	}
	return nil
}

func (w *walMergeDocs) WriteToShardRollback(shard *IndexShard, shardID int64, batch *blugeindex.Batch) error {
	docs, ok := (*w)[shardID]
	if !ok {
		return nil
	}
	var writer *bluge.Writer
	var err error
	if shardID >= 0 {
		writer, err = shard.GetWriter(shardID)
	} else {
		return nil // no insert
	}
	if err != nil {
		return err
	}
	var firstAction string
	for _, doc := range docs {
		bdoc, err := shard.BuildBlugeDocumentFromJSON(doc.docID, doc.data)
		if err != nil {
			return err
		}
		firstAction = doc.actions[0]
		switch firstAction {
		case meta.ActionTypeInsert:
			batch.Delete(bdoc.ID())
		case meta.ActionTypeUpdate:
			// skip
		case meta.ActionTypeDelete:
			// skip
		}
	}

	return writer.Batch(batch)
}

// parseInterval parse interval string to time.Duration: 1s, 10ms
func parseInterval(v string) (time.Duration, error) {
	if v == "" {
		return time.Second, nil
	}
	v = strings.ToLower(v)
	if strings.HasSuffix(v, "ms") {
		i, err := strconv.Atoi(v[:len(v)-2])
		return time.Millisecond * time.Duration(i), err
	}
	if strings.HasSuffix(v, "s") {
		i, err := strconv.Atoi(v[:len(v)-1])
		return time.Second * time.Duration(i), err
	}
	i, err := strconv.Atoi(v)
	return time.Second * time.Duration(i), err
}
