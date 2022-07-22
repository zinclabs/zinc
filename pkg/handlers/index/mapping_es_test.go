package index

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/test/utils"
)

func TestESMapping_GetConverted(t *testing.T) {
	config.Global.EnableTextKeywordMapping = true

	t.Run("create index", func(t *testing.T) {
		index, err := core.NewIndex("TestEsMapping.index_1", "disk", 2)
		assert.NoError(t, err)
		assert.NotNil(t, index)

		err = core.StoreIndex(index)
		assert.NoError(t, err)
	})

	t.Run("set mapping", func(t *testing.T) {
		type args struct {
			code    int
			data    map[string]interface{}
			rawData string
			target  string
			result  string
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{
				name: "normal",
				args: args{
					code: http.StatusOK,
					data: map[string]interface{}{
						"properties": map[string]interface{}{
							"Athlete": map[string]interface{}{
								"type":          "text",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  false,
								"highlightable": false,
							},
							"City": map[string]interface{}{
								"type":          "keyword",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
							"Gender": map[string]interface{}{
								"type":          "bool",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
							"time": map[string]interface{}{
								"type":          "date",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
							"obj": map[string]interface{}{
								"type":          "text",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
							"obj.sub_field": map[string]interface{}{
								"type":          "text",
								"index":         true,
								"store":         false,
								"sortable":      false,
								"aggregatable":  true,
								"highlightable": false,
							},
						},
					},
					target: "TestEsMapping.index_1",
					result: `{"message":"ok"}`,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c, w := utils.NewGinContext()
				if tt.args.data != nil {
					utils.SetGinRequestData(c, tt.args.data)
				}
				if tt.args.rawData != "" {
					utils.SetGinRequestData(c, tt.args.rawData)
				}
				utils.SetGinRequestParams(c, map[string]string{"target": tt.args.target})
				SetMapping(c)
				assert.Equal(t, tt.args.code, w.Code)
				assert.Equal(t, tt.args.result, w.Body.String())
			})
		}
	})

	t.Run("get mapping", func(t *testing.T) {
		type args struct {
			code   int
			target string
			result string
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{
				name: "normal",
				args: args{
					code:   http.StatusOK,
					target: "TestEsMapping.index_1",
					result: `{"@timestamp":{"type":"date"}`,
				},
				wantErr: false,
			},
			{
				name: "empty",
				args: args{
					code:   http.StatusBadRequest,
					target: "",
					result: `{"error":"index  does not exists"}`,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c, w := utils.NewGinContext()
				utils.SetGinRequestParams(c, map[string]string{"target": tt.args.target})
				GetESMapping(c)
				assert.Equal(t, tt.args.code, w.Code)
				assert.Contains(t, w.Body.String(), tt.args.result)
			})
		}
	})

	t.Run("delete index", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_ = core.DeleteIndex(fmt.Sprintf("TestEsMapping.index_%d", i))
		}
	})

	config.Global.EnableTextKeywordMapping = false
}
