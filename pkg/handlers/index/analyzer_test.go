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

package index

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/test/utils"
)

func TestAnalyze(t *testing.T) {
	indexName := "TestAnalyze.index_1"
	type args struct {
		code   int
		data   string
		params map[string]string
		result string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "use default analyzer",
			args: args{
				code:   http.StatusOK,
				data:   `{"analyzer":"standard","text":"this is a test 2022 year"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test 2022 year]",
			},
			wantErr: false,
		},
		{
			name: "use default analyzer",
			args: args{
				code:   http.StatusOK,
				data:   `{"analyzer":"standard","text":"这是来自2012年的测试"}`,
				params: map[string]string{"target": ""},
				result: "[这 是 来 自 2012 年 的 测 试]",
			},
			wantErr: false,
		},
		{
			name: "with index analyzer",
			args: args{
				code:   http.StatusOK,
				data:   `{"analyzer":"standard","text":"this is a test"}`,
				params: map[string]string{"target": indexName},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "with index not exist analyzer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"analyzer":"standardNoExists","text":"this is a test"}`,
				params: map[string]string{"target": indexName},
				result: "[this is a test]",
			},
			wantErr: true,
		},
		{
			name: "with index empty analyzer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"analyzer":"","text":"this is a test"}`,
				params: map[string]string{"target": indexName},
				result: "[this is a test]",
			},
			wantErr: true,
		},
		{
			name: "with index field analyzer",
			args: args{
				code:   http.StatusOK,
				data:   `{"field":"name1","text":"this is a test"}`,
				params: map[string]string{"target": indexName},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "with index field analyzer",
			args: args{
				code:   http.StatusOK,
				data:   `{"field":"name2","text":"this is a test"}`,
				params: map[string]string{"target": indexName},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "with not exist index analyzer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"analyzer":"standard","text":"this is a test"}`,
				params: map[string]string{"target": "not_exist_index"},
				result: "[this is a test]",
			},
			wantErr: true,
		},
		{
			name: "with empty analyzer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"analyzer":"","text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test]",
			},
			wantErr: true,
		},
		{
			name: "with not exist analyzer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"analyzer":"standardNoExist","text":"this is a test"}`,
				params: map[string]string{"target": ""},
			},
			wantErr: true,
		},
		{
			name: "with error json",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"analyzer":"standard","text":"this is a test"x}`,
				params: map[string]string{"target": indexName},
			},
			wantErr: true,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusOK,
				data:   `{"tokenizer":["standard"],"token_filter":["camel_case"],"char_filter":["html_strip"],"text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusOK,
				data:   `{"tokenizer":["standard"],"filter":["camel_case"],"char_filter":["html_strip"],"text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusOK,
				data:   `{"tokenizer":"standard","filter":"camel_case","char_filter":"html_strip","text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusOK,
				data:   `{"tokenizer":{"type":"standard"},"filter":{"type":"camel_case"},"char_filter":{"type":"html_strip"},"text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusOK,
				data:   `{"tokenizer":{"cu":{"type":"standard"}},"filter":{"cu":{"type":"camel_case"}},"char_filter":{"cu":{"type":"html_strip"}},"text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "[this is a test]",
			},
			wantErr: false,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"tokenizer":"standard","token_filter":"camel_case","char_filter":1,"text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "",
			},
			wantErr: true,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"tokenizer":"standard","token_filter":1,"char_filter":"html_strip","text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "",
			},
			wantErr: true,
		},
		{
			name: "empty analyzer with custom tokenizer",
			args: args{
				code:   http.StatusBadRequest,
				data:   `{"tokenizer":1,"token_filter":"camel_case","char_filter":"html_strip","text":"this is a test"}`,
				params: map[string]string{"target": ""},
				result: "",
			},
			wantErr: true,
		},
	}

	t.Run("prepare", func(t *testing.T) {
		index, err := core.NewIndex(indexName, "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		mapping := meta.NewMappings()
		prop1 := meta.NewProperty("text")
		prop1.Analyzer = "standard"
		mapping.Properties["name1"] = prop1
		prop2 := meta.NewProperty("text")
		prop2.Analyzer = "standard"
		prop2.SearchAnalyzer = "standard"
		mapping.Properties["name2"] = prop2
		err = index.SetMappings(mapping)
		assert.NoError(t, err)

		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := utils.NewGinContext()
			utils.SetGinRequestData(c, tt.args.data)
			utils.SetGinRequestParams(c, tt.args.params)
			Analyze(c)
			assert.Equal(t, tt.args.code, w.Code)
			tokens, err := getTokenStrings(w.Body.Bytes())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.result, tokens)
			}
		})
	}

	t.Run("cleanup", func(t *testing.T) {
		err := core.DeleteIndex(indexName)
		assert.NoError(t, err)
	})
}

func getTokenStrings(data []byte) (string, error) {
	var ret map[string]interface{}
	err := json.Unmarshal(data, &ret)
	if err != nil {
		return "", err
	}

	tokens, _ := ret["tokens"].([]interface{})
	if tokens == nil {
		return "", fmt.Errorf("tokens not exists")
	}

	strs := make([]string, 0, len(tokens))
	for _, token := range tokens {
		str := token.(map[string]interface{})["token"].(string)
		strs = append(strs, str)
	}

	return "[" + strings.Join(strs, " ") + "]", nil
}
