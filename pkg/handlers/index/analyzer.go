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

	"github.com/blugelabs/bluge/analysis"
	"github.com/gin-gonic/gin"

	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
	zincanalysis "github.com/zinclabs/zinc/pkg/uquery/analysis"
	"github.com/zinclabs/zinc/pkg/zutils"
)

// @Id Analyze
// @Summary Analyze
// @Tags    Index
// @Accept  json
// @Produce json
// @Param   query  body  object  true  "Query"
// @Success 200 {object} AnalyzeResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Router /api/_analyze [post]
func Analyze(c *gin.Context) {
	var query AnalyzeRequest
	if err := zutils.GinBindJSON(c, &query); err != nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: err.Error()})
		return
	}

	var err error
	var ana *analysis.Analyzer
	indexName := c.Param("target")
	if indexName != "" {
		// use index analyzer
		index, exists := core.GetIndex(indexName)
		if !exists {
			c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index " + indexName + " does not exists"})
			return
		}
		if query.Filed != "" && query.Analyzer == "" {
			mappings := index.GetMappings()
			if mappings != nil && mappings.Len() > 0 {
				if prop, ok := mappings.GetProperty(query.Filed); ok {
					if query.Analyzer == "" && prop.SearchAnalyzer != "" {
						query.Analyzer = prop.SearchAnalyzer
					}
					if query.Analyzer == "" && prop.Analyzer != "" {
						query.Analyzer = prop.Analyzer
					}
				}
			}
		}
		ana, _ = zincanalysis.QueryAnalyzer(index.GetAnalyzers(), query.Analyzer)
		if ana == nil {
			if query.Analyzer == "" {
				ana = new(analysis.Analyzer)
			} else {
				c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "analyzer " + query.Analyzer + " does not exists"})
				return
			}
		}
	} else {
		// none index specified
		ana, _ = zincanalysis.QueryAnalyzer(nil, query.Analyzer)
		if ana == nil {
			if query.Analyzer == "" {
				ana = new(analysis.Analyzer)
			} else {
				c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "analyzer " + query.Analyzer + " does not exists"})
				return
			}
		}
	}

	charFilters, err := parseCharFilter(query.CharFilter)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	if query.TokenFilter == nil && query.Filter != nil {
		query.TokenFilter = query.Filter
		query.Filter = nil
	}
	tokenFilters, err := parseTokenFilter(query.TokenFilter)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	tokenizers, err := parseTokenizer(query.Tokenizer)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	if len(charFilters) > 0 {
		ana.CharFilters = append(ana.CharFilters, charFilters...)
	}

	if len(tokenFilters) > 0 {
		ana.TokenFilters = append(ana.TokenFilters, tokenFilters...)
	}

	if len(tokenizers) > 0 {
		ana.Tokenizer = tokenizers[0]
	}

	if ana.Tokenizer == nil {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "analyzer need set a tokenizer"})
		return
	}

	tokens := ana.Analyze([]byte(query.Text))
	ret := AnalyzeResponse{}
	ret.Tokens = make([]AnalyzeResponseToken, 0, len(tokens))
	for _, token := range tokens {
		ret.Tokens = append(ret.Tokens, formatToken(token))
	}
	c.JSON(http.StatusOK, ret)
}

// @Id AnalyzeIndex
// @Summary Analyze
// @Tags    Index
// @Accept  json
// @Produce json
// @Param   index  path  string  true  "Index"
// @Param   query  body  object  true  "Query"
// @Success 200 {object} AnalyzeResponse
// @Failure 400 {object} meta.HTTPResponseError
// @Router /api/{index}/_analyze [post]
func AnalyzeIndexForSDK() {}

func parseTokenizer(data interface{}) ([]analysis.Tokenizer, error) {
	if data == nil {
		return nil, nil
	}

	tokenizers := make([]analysis.Tokenizer, 0)
	switch v := data.(type) {
	case string:
		zer, err := zincanalysis.RequestTokenizerSingle(v, nil)
		if err != nil {
			return nil, err
		}
		tokenizers = append(tokenizers, zer)
	case []interface{}:
		zers, err := zincanalysis.RequestTokenizerSlice(v)
		if err != nil {
			return nil, err
		}
		tokenizers = append(tokenizers, zers...)
	case map[string]interface{}:
		typ, err := zutils.GetStringFromMap(v, "type")
		if typ != "" && err == nil {
			zer, err := zincanalysis.RequestTokenizerSingle(typ, v)
			if err != nil {
				return nil, err
			}
			tokenizers = append(tokenizers, zer)
		} else {
			zers, err := zincanalysis.RequestTokenizer(v)
			if err != nil {
				return nil, err
			}
			for _, zer := range zers {
				tokenizers = append(tokenizers, zer)
			}
		}
	default:
		return nil, fmt.Errorf("tokenizer unsuported type")
	}

	return tokenizers, nil
}

func parseTokenFilter(data interface{}) ([]analysis.TokenFilter, error) {
	if data == nil {
		return nil, nil
	}

	tokens := make([]analysis.TokenFilter, 0)
	switch v := data.(type) {
	case string:
		filter, err := zincanalysis.RequestTokenFilterSingle(v, nil)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, filter)
	case []interface{}:
		filters, err := zincanalysis.RequestTokenFilterSlice(v)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, filters...)
	case map[string]interface{}:
		typ, err := zutils.GetStringFromMap(v, "type")
		if typ != "" && err == nil {
			filter, err := zincanalysis.RequestTokenFilterSingle(typ, v)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, filter)
		} else {
			filters, err := zincanalysis.RequestTokenFilter(v)
			if err != nil {
				return nil, err
			}
			for _, filter := range filters {
				tokens = append(tokens, filter)
			}
		}
	default:
		return nil, fmt.Errorf("token_filter unsuported type")
	}

	return tokens, nil
}

func parseCharFilter(data interface{}) ([]analysis.CharFilter, error) {
	if data == nil {
		return nil, nil
	}

	chars := make([]analysis.CharFilter, 0)
	switch v := data.(type) {
	case string:
		filter, err := zincanalysis.RequestCharFilterSingle(v, nil)
		if err != nil {
			return nil, err
		}
		chars = append(chars, filter)
	case []interface{}:
		filters, err := zincanalysis.RequestCharFilterSlice(v)
		if err != nil {
			return nil, err
		}
		chars = append(chars, filters...)
	case map[string]interface{}:
		typ, err := zutils.GetStringFromMap(v, "type")
		if typ != "" && err == nil {
			filter, err := zincanalysis.RequestCharFilterSingle(typ, v)
			if err != nil {
				return nil, err
			}
			chars = append(chars, filter)
		} else {
			filters, err := zincanalysis.RequestCharFilter(v)
			if err != nil {
				return nil, err
			}
			for _, filter := range filters {
				chars = append(chars, filter)
			}
		}
	default:
		return nil, fmt.Errorf("char_filter unsuported type")
	}

	return chars, nil
}

func formatToken(token *analysis.Token) AnalyzeResponseToken {
	return AnalyzeResponseToken{
		Token:       string(token.Term),
		StartOffset: token.Start,
		EndOffset:   token.End,
		Position:    token.PositionIncr,
		Type:        formatTokenType(token.Type),
		Keyword:     token.KeyWord,
	}
}

func formatTokenType(typ analysis.TokenType) string {
	switch typ {
	case analysis.AlphaNumeric:
		return "AlphaNumeric"
	case analysis.Ideographic:
		return "Ideographic"
	case analysis.Numeric:
		return "Numeric"
	case analysis.DateTime:
		return "DateTime"
	case analysis.Shingle:
		return "Shingle"
	case analysis.Single:
		return "Single"
	case analysis.Double:
		return "Double"
	case analysis.Boolean:
		return "Boolean"
	default:
		return "Unknown"
	}
}

type AnalyzeRequest struct {
	Analyzer    string      `json:"analyzer"`
	Filed       string      `json:"field"`
	Text        string      `json:"text"`
	Tokenizer   interface{} `json:"tokenizer"`
	CharFilter  interface{} `json:"char_filter"`
	TokenFilter interface{} `json:"token_filter"`
	Filter      interface{} `json:"filter"` // compatibility with es, alias for TokenFilter
}

type AnalyzeResponse struct {
	Tokens []AnalyzeResponseToken `json:"tokens"`
}

type AnalyzeResponseToken struct {
	Token       string `json:"token"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Position    int    `json:"position"`
	Type        string `json:"type"`
	Keyword     bool   `json:"keyword"`
}
