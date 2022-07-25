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

	"github.com/zinclabs/zinc/pkg/meta"
)

func TestListTemplates(t *testing.T) {
	type args struct {
		pattern string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "all",
			args: args{
				pattern: "",
			},
			want: 3,
		},
		{
			name: "pattern1",
			args: args{
				pattern: "TestListTemplates-log-*",
			},
			want: 1,
		},
		{
			name: "pattern2",
			args: args{
				pattern: "TestListTemplates-error-*",
			},
			want: 1,
		},
		{
			name: "pattern3",
			args: args{
				pattern: "TestListTemplates-log-error-*",
			},
			want: 1,
		},
		{
			name: "not found",
			args: args{
				pattern: "TestListTemplates-logNot-*",
			},
			want: 0,
		},
	}

	templates := map[string]*meta.IndexTemplate{
		"log": {
			IndexPatterns: []string{"TestListTemplates-log-*"},
			Priority:      100,
		},
		"error": {
			IndexPatterns: []string{"TestListTemplates-error-*"},
			Priority:      100,
		},
		"log-error": {
			IndexPatterns: []string{"TestListTemplates-log-error-*"},
			Priority:      200,
		},
	}

	t.Run("prepare", func(t *testing.T) {
		for name, tpl := range templates {
			err := NewTemplate(name, tpl)
			assert.NoError(t, err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListTemplates(tt.args.pattern)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, len(got))
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		for name := range templates {
			err := DeleteTemplate(name)
			assert.NoError(t, err)
		}

		tpls, err := ListTemplates("")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(tpls))
	})
}

func TestNewTemplate(t *testing.T) {
	type args struct {
		name     string
		template *meta.IndexTemplate
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "log",
			args: args{
				name: "log",
				template: &meta.IndexTemplate{
					IndexPatterns: []string{"TestNewTemplate-log-*"},
					Priority:      100,
				},
			},
		},
		{
			name: "error",
			args: args{
				name: "error",
				template: &meta.IndexTemplate{
					IndexPatterns: []string{"TestNewTemplate-error-*", "TestNewTemplate-log-error-*"},
					Priority:      100,
				},
			},
		},
		{
			name: "with same name",
			args: args{
				name: "error",
				template: &meta.IndexTemplate{
					IndexPatterns: []string{"TestNewTemplate-error-*"},
					Priority:      199,
				},
			},
			wantErr: false,
		},
		{
			name: "with different priority",
			args: args{
				name: "error3",
				template: &meta.IndexTemplate{
					IndexPatterns: []string{"TestNewTemplate-error-*"},
					Priority:      199,
				},
			},
			wantErr: false,
		},
		{
			name: "with same priority",
			args: args{
				name: "error4",
				template: &meta.IndexTemplate{
					IndexPatterns: []string{"TestNewTemplate-error-*"},
					Priority:      199,
				},
			},
			wantErr: true,
		},
		{
			name: "nil",
			args: args{
				name:     "error5",
				template: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTemplate(tt.args.name, tt.args.template)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := DeleteTemplate(tt.args.name)
				assert.NoError(t, err)
			})
		}

		tpls, err := ListTemplates("")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(tpls))
	})
}

func TestLoadTemplate(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   bool
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				name: "log",
			},
			want:  true,
			want1: true,
		},
		{
			name: "empty",
			args: args{
				name: "",
			},
			want:  false,
			want1: false,
		},
	}

	templates := map[string]*meta.IndexTemplate{
		"log": {
			IndexPatterns: []string{"TestLoadTemplate-log-*"},
			Priority:      100,
		},
		"error": {
			IndexPatterns: []string{"TestLoadTemplate-error-*"},
			Priority:      100,
		},
		"log-error": {
			IndexPatterns: []string{"TestLoadTemplate-log-error-*"},
			Priority:      200,
		},
	}

	t.Run("prepare", func(t *testing.T) {
		for name, tpl := range templates {
			err := NewTemplate(name, tpl)
			assert.NoError(t, err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := LoadTemplate(tt.args.name)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			if tt.want {
				assert.NotNil(t, got)
			}
			assert.Equal(t, tt.want1, got1)
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		for name := range templates {
			err := DeleteTemplate(name)
			assert.NoError(t, err)
		}

		tpls, err := ListTemplates("")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(tpls))
	})
}

func TestDeleteTemplate(t *testing.T) {
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
				name: "normal",
			},
		},
		{
			name: "empty",
			args: args{
				name: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteTemplate(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("DeleteTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUseTemplate(t *testing.T) {
	type args struct {
		indexName string
	}
	tests := []struct {
		name         string
		args         args
		want         bool
		wantPriority int
		wantErr      bool
	}{
		{
			name: "normal",
			args: args{
				indexName: "TestUseTemplate-log-2022.02.02",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "not exits",
			args: args{
				indexName: "TestUseTemplate-No-2022.02.02",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "with priority1",
			args: args{
				indexName: "TestUseTemplate-error-2022.02.02",
			},
			want:         true,
			wantPriority: 200,
			wantErr:      false,
		},
		{
			name: "with priority2",
			args: args{
				indexName: "TestUseTemplate-log-error-2022.02.02",
			},
			want:         true,
			wantPriority: 200,
			wantErr:      false,
		},
	}

	templates := map[string]*meta.IndexTemplate{
		"log": {
			IndexPatterns: []string{"TestUseTemplate-log-*"},
			Priority:      100,
		},
		"error": {
			IndexPatterns: []string{"TestUseTemplate-error-*"},
			Priority:      100,
		},
		"errorHighPriority": {
			IndexPatterns: []string{"TestUseTemplate-error-*"},
			Priority:      200,
		},
		"log-error": {
			IndexPatterns: []string{"TestUseTemplate-log-error-*"},
			Priority:      200,
			Template: meta.TemplateTemplate{
				Settings: &meta.IndexSettings{
					NumberOfShards: 3,
				},
				Mappings: &meta.Mappings{
					Properties: map[string]meta.Property{
						"name": {
							Type:  "text",
							Index: true,
						},
					},
				},
			},
		},
	}

	t.Run("prepare", func(t *testing.T) {
		for name, tpl := range templates {
			err := NewTemplate(name, tpl)
			assert.NoError(t, err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UseTemplate(tt.args.indexName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.want {
				assert.NotNil(t, got)
				if tt.wantPriority > 0 {
					assert.Equal(t, tt.wantPriority, got.Priority)
				}
			}

			t.Run("new index use template", func(t *testing.T) {
				indexName := "TestUseTemplate-log-error-2022.02.02"
				index, err := NewIndex(indexName, "", 1)
				assert.NoError(t, err)
				assert.NotNil(t, index)

				err = StoreIndex(index)
				assert.NoError(t, err)

				err = DeleteIndex(indexName)
				assert.NoError(t, err)
			})
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		for name := range templates {
			err := DeleteTemplate(name)
			assert.NoError(t, err)
		}

		tpls, err := ListTemplates("")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(tpls))
	})
}
