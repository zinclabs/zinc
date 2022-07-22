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

package wal

import (
	"path"

	"github.com/zinclabs/wal"
	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/wal/redo"
)

type Log struct {
	name string
	log  *wal.Log
	Redo *redo.Log
}

func Open(indexName string) (*Log, error) {
	var err error
	l := new(Log)
	l.name = indexName
	opt := &wal.Options{
		NoSync:           true,       // Fsync after every write
		SegmentSize:      16777216,   // 16 MB log segment files.
		LogFormat:        wal.Binary, // Binary format is small and fast.
		SegmentCacheSize: 2,          // Number of cached in-memory segments
		NoCopy:           true,       // Make a new copy of data for every Read call.
		DirPerms:         0750,       // Permissions for the created directories
		FilePerms:        0640,       // Permissions for the created data files
		FillID:           true,       // Allow writes with a zero ID
	}
	l.log, err = wal.Open(path.Join(config.Global.DataPath, indexName, "wal"), opt)
	if err != nil {
		return nil, errors.New(errors.ErrorTypeRuntimeException, "open wal error").Cause(err)
	}

	redoOpt := redo.DefaultOptions()
	redoOpt.NoSync = config.Global.WalRedoLogNoSync
	redoOpt.NoCopy = true
	l.Redo, err = redo.Open(path.Join(config.Global.DataPath, indexName, "redo"), redoOpt)
	if err != nil {
		return nil, errors.New(errors.ErrorTypeRuntimeException, "open wal redo error").Cause(err)
	}

	return l, nil
}

func (l *Log) Name() string {
	return l.name
}

func (l *Log) Len() (uint64, error) {
	return l.log.Len()
}

func (l *Log) FirstIndex() (uint64, error) {
	return l.log.FirstIndex()
}

func (l *Log) LastIndex() (uint64, error) {
	return l.log.LastIndex()
}

func (l *Log) Write(data []byte) error {
	return l.log.Write(0, data)
}

func (l *Log) Read(id uint64) ([]byte, error) {
	return l.log.Read(id)
}

func (l *Log) TruncateFront(id uint64) error {
	return l.log.TruncateFront(id)
}

func (l *Log) Sync() error {
	return l.log.Sync()
}

func (l *Log) Close() error {
	if err := l.log.Close(); err != nil {
		return err
	}
	return l.Redo.Close()
}
