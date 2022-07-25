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

	"github.com/stretchr/testify/assert"
)

func TestIndexList_List(t *testing.T) {
	indexName := "TestIndexList_List.index_1"
	index, exist, err := GetOrCreateIndex(indexName, "", 2)
	assert.NoError(t, err)
	assert.False(t, exist)
	assert.NotNil(t, index)

	rs, err := index.GetReaders(0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, rs)

	got1 := ZINC_INDEX_LIST.List()
	assert.NotNil(t, got1)

	got2 := ZINC_INDEX_LIST.ListName()
	assert.NotNil(t, got2)

	got3 := ZINC_INDEX_LIST.ListStat()
	assert.NotNil(t, got3)

	err = DeleteIndex(indexName)
	assert.NoError(t, err)

	err = ZINC_INDEX_LIST.GC()
	assert.NoError(t, err)
}
