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

func TestTelemetry(t *testing.T) {
	indexName := "TestTelemetry.index_1"
	t.Run("prepare", func(t *testing.T) {
		index, err := NewIndex(indexName, "disk", 1)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("telemetry", func(t *testing.T) {
		id := Telemetry.createInstanceID()
		assert.NotEmpty(t, id)
		Telemetry.Instance()
		Telemetry.Event("server_start", nil)
		Telemetry.Cron()

		Telemetry.GetIndexSize(indexName)
		Telemetry.HeartBeat()
	})

	t.Run("cleanup", func(t *testing.T) {
		err := DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}
