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

func TestNewIndex(t *testing.T) {
	type args struct {
		name        string
		storageType string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				name:        "TestNewIndex.index_1",
				storageType: "disk",
			},
			wantErr: false,
		},
		{
			name: "underline prefix",
			args: args{
				name:        "_TestNewIndex.index_2",
				storageType: "disk",
			},
			wantErr: true,
		},
		{
			name: "with analyzer",
			args: args{
				name:        "TestNewIndex.index_3",
				storageType: "disk",
			},
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				name:        "",
				storageType: "disk",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIndex(tt.args.name, tt.args.storageType, 2)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, got)

			err = StoreIndex(got)
			assert.NoError(t, err)

			t.Run("cleanup", func(t *testing.T) {
				err := DeleteIndex(tt.args.name)
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetIndex(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name  string
		args  args
		want  *Index
		want1 bool
	}{
		{
			name: "normal",
			args: args{
				name: "TestGetIndex.index_1",
			},
			want1: true,
		},
		{
			name: "not exist",
			args: args{
				name: "TestGetIndex.index_2",
			},
			want1: false,
		},
	}

	t.Run("prepare", func(t *testing.T) {
		index, err := NewIndex("TestGetIndex.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = StoreIndex(index)
		assert.NoError(t, err)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetIndex(tt.args.name)
			if !tt.want1 {
				assert.False(t, got1)
				return
			}
			assert.True(t, got1)
			assert.NotNil(t, got)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err := DeleteIndex("TestGetIndex.index_1")
		assert.NoError(t, err)
	})
}

func TestCheckIndexName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				name: "TestCheckIndexName.index_1",
			},
		},
		{
			name: "error",
			args: args{
				name: "_TestCheckIndexName.index_1",
			},
			wantErr: true,
		},
		{
			name: "error",
			args: args{
				name: "_TestCheckIndexName.index_*",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckIndexName(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("CheckIndexName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
