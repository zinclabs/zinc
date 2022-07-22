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

package metadata

import (
	"github.com/goccy/go-json"

	"github.com/zinclabs/zinc/pkg/meta"
)

type index struct{}

var Index = new(index)

func (t *index) List(offset, limit int) ([]*meta.Index, error) {
	data, err := db.List(t.key(""), offset, limit)
	if err != nil {
		return nil, err
	}
	indexes := make([]*meta.Index, 0, len(data))
	for _, d := range data {
		idx := new(meta.Index)
		err = json.Unmarshal(d, idx)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}
	return indexes, nil
}

func (t *index) Get(id string) (*meta.Index, error) {
	data, err := db.Get(t.key(id))
	if err != nil {
		return nil, err
	}
	idx := new(meta.Index)
	err = json.Unmarshal(data, idx)
	return idx, err
}

func (t *index) Set(id string, data []byte) error {
	return db.Set(t.key(id), data)
}

func (t *index) Delete(id string) error {
	return db.Delete(t.key(id))
}

func (t *index) key(id string) string {
	return "/index/" + id
}
