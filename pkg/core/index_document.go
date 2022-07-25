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
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
)

// CreateDocument inserts or updates a document in the zinc index
func (index *Index) CreateDocument(docID string, doc map[string]interface{}, update bool) error {
	shard := index.GetShardByDocID(docID)

	// check WAL
	if err := shard.OpenWAL(); err != nil {
		return err
	}

	secondShardID := shard.GetLatestShardID()
	if update {
		secondShardID = -1
	}
	data, err := shard.CheckDocument(docID, doc, update, secondShardID)
	if err != nil {
		return err
	}

	return shard.wal.Write(data)
}

// UpdateDocument updates a document in the zinc index
func (index *Index) UpdateDocument(docID string, doc map[string]interface{}, insert bool) error {
	shard := index.GetShardByDocID(docID)

	// check WAL
	if err := shard.OpenWAL(); err != nil {
		return err
	}

	update := true
	secondShardID, err := shard.FindShardByDocID(docID)
	if err != nil {
		if insert && err == errors.ErrorIDNotFound {
			update = false
		} else {
			return err
		}
	}

	data, err := shard.CheckDocument(docID, doc, update, secondShardID)
	if err != nil {
		return err
	}

	return shard.wal.Write(data)
}

// DeleteDocument deletes a document in the zinc index
func (index *Index) DeleteDocument(docID string) error {
	shard := index.GetShardByDocID(docID)

	// check WAL
	if err := shard.OpenWAL(); err != nil {
		return err
	}

	secondShardID, err := shard.FindShardByDocID(docID)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		meta.IDFieldName:     docID,
		meta.ActionFieldName: meta.ActionTypeDelete,
		meta.ShardFieldName:  secondShardID,
	}
	jstr, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return shard.wal.Write(jstr)
}

// isDateProperty returns true if the given value matches the default date format.
func isDateProperty(value string) (string, bool) {
	layout := detectTimeLayout(value)
	if layout == "" {
		return "", false
	}
	_, err := time.Parse(layout, value)
	return layout, err == nil
}

// detectTimeLayout tries to figure out the correct layout of the input date.
func detectTimeLayout(value string) string {
	layout := ""
	switch {
	case len(value) == 19 && strings.Index(value, " ") == 10:
		layout = "2006-01-02 15:04:05"
	case len(value) == 19 && strings.Index(value, "T") == 10:
		layout = "2006-01-02T15:04:05"
	case len(value) == 25 && strings.Index(value, "T") == 10:
		layout = time.RFC3339
	case len(value) == 29 && strings.Index(value, "T") == 10 && strings.Index(value, ".") == 19:
		layout = "2006-01-02T15:04:05.999Z07:00"
	}

	return layout
}
