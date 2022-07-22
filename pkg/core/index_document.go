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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/goccy/go-json"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
	zincanalysis "github.com/zinclabs/zinc/pkg/uquery/analysis"
	"github.com/zinclabs/zinc/pkg/zutils"
	"github.com/zinclabs/zinc/pkg/zutils/flatten"
)

// BuildBlugeDocumentFromJSON returns the bluge document for the json document. It also updates the mapping for the fields if not found.
// If no mappings are found, it creates te mapping for all the encountered fields. If mapping for some fields is found but not for others
// then it creates the mapping for the missing fields.
func (index *Index) BuildBlugeDocumentFromJSON(docID string, doc map[string]interface{}) (*bluge.Document, error) {
	// Pick the index mapping from the cache if it already exists
	mappings := index.GetMappings()

	delete(doc, meta.ActionFieldName)
	delete(doc, meta.IDFieldName)
	delete(doc, meta.ShardFieldName)

	// Create a new bluge document
	bdoc := bluge.NewDocument(docID)
	// Iterate through each field and add it to the bluge document
	for key, value := range doc {
		if value == nil || key == meta.TimeFieldName {
			continue
		}

		prop, ok := mappings.GetProperty(key)
		if !ok || !prop.Index {
			continue // not index, skip
		}

		switch v := value.(type) {
		case []interface{}:
			for _, v := range v {
				if err := index.buildField(mappings, bdoc, key, v); err != nil {
					return nil, err
				}
			}
		default:
			if err := index.buildField(mappings, bdoc, key, v); err != nil {
				return nil, err
			}
		}
	}

	// set timestamp
	timestamp := time.Now()
	if value, ok := doc[meta.TimeFieldName]; ok {
		delete(doc, meta.TimeFieldName)
		timestamp = time.Unix(0, int64(value.(float64)))
	}
	bdoc.AddField(bluge.NewDateTimeField(meta.TimeFieldName, timestamp).StoreValue().Sortable().Aggregatable())

	docByteVal, _ := json.Marshal(doc)
	bdoc.AddField(bluge.NewStoredOnlyField("_index", []byte(index.GetName())))
	bdoc.AddField(bluge.NewStoredOnlyField("_source", docByteVal))
	bdoc.AddField(bluge.NewCompositeFieldExcluding("_all", []string{"_id", "_index", "_source", meta.TimeFieldName}))

	// Add time for index
	bdoc.SetTimestamp(timestamp.UnixNano())
	// Upate metadata
	index.SetTimestamp(docID, timestamp.UnixNano())

	return bdoc, nil
}

func (index *Index) buildField(mappings *meta.Mappings, bdoc *bluge.Document, key string, value interface{}) error {
	var field *bluge.TermField
	prop, _ := mappings.GetProperty(key)
	switch prop.Type {
	case "text":
		field = bluge.NewTextField(key, value.(string)).SearchTermPositions()
		fieldAnalyzer, _ := zincanalysis.QueryAnalyzerForField(index.GetAnalyzers(), mappings, key)
		if fieldAnalyzer != nil {
			field.WithAnalyzer(fieldAnalyzer)
		}
	case "numeric":
		field = bluge.NewNumericField(key, value.(float64))
	case "keyword":
		field = bluge.NewKeywordField(key, value.(string))
	case "bool":
		field = bluge.NewKeywordField(key, strconv.FormatBool(value.(bool)))
	case "date", "time":
		v, err := zutils.ParseTime(value, prop.Format, prop.TimeZone)
		if err != nil {
			return fmt.Errorf("field [%s] value [%v] parse err: %s", key, value, err.Error())
		}
		field = bluge.NewDateTimeField(key, v)
	}
	if prop.Store || prop.Highlightable {
		field.StoreValue()
	}
	if prop.Highlightable {
		field.HighlightMatches()
	}
	if prop.Sortable {
		field.Sortable()
	}
	if prop.Aggregatable {
		field.Aggregatable()
	}
	bdoc.AddField(field)

	if prop.Fields != nil {
		for propField := range prop.Fields {
			err := index.buildField(mappings, bdoc, key+"."+propField, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CheckDocument checks if the document is valid.
func (index *Index) CheckDocument(docID string, doc map[string]interface{}, update bool, shard int64) ([]byte, error) {
	// Pick the index mapping from the cache if it already exists
	mappings := index.GetMappings()

	mappingsNeedsUpdate := false

	flatDoc, _ := flatten.Flatten(doc, "")
	// Iterate through each field and add it to the bluge document
	for key, value := range flatDoc {
		if value == nil {
			continue
		}

		if _, ok := mappings.GetProperty(key); !ok {
			// try to find the type of the value and use it to define default mapping
			switch v := value.(type) {
			case string:
				if layout, ok := isDateProperty(v); ok {
					prop := meta.NewProperty("date")
					prop.Format = layout
					mappings.SetProperty(key, prop)
				} else {
					newProp := meta.NewProperty("text")
					if config.Global.EnableTextKeywordMapping {
						p := meta.NewProperty("keyword")
						newProp.AddField("keyword", p)
						mappings.SetProperty(key+".keyword", p)
					}
					mappings.SetProperty(key, newProp)
				}
			case int, int64, float64:
				mappings.SetProperty(key, meta.NewProperty("numeric"))
			case bool:
				mappings.SetProperty(key, meta.NewProperty("bool"))
			case []interface{}:
				if v, ok := value.([]interface{}); ok {
					for _, vv := range v {
						switch val := vv.(type) {
						case string:
							if layout, ok := isDateProperty(val); ok {
								prop := meta.NewProperty("date")
								prop.Format = layout
								mappings.SetProperty(key, prop)
							} else {
								newProp := meta.NewProperty("text")
								if config.Global.EnableTextKeywordMapping {
									p := meta.NewProperty("keyword")
									newProp.AddField("keyword", p)
									mappings.SetProperty(key+".keyword", p)
								}
								mappings.SetProperty(key, newProp)
							}
						case float64:
							mappings.SetProperty(key, meta.NewProperty("numeric"))
						case bool:
							mappings.SetProperty(key, meta.NewProperty("bool"))
						}
						break
					}
				}
			}

			mappingsNeedsUpdate = true
		}

		prop, ok := mappings.GetProperty(key)
		if !ok || !prop.Index {
			continue // not index, skip
		}

		switch v := value.(type) {
		case []interface{}:
			for i, v := range v {
				if err := index.checkField(mappings, flatDoc, key, v, i, true); err != nil {
					return nil, err
				}
			}
		default:
			if err := index.checkField(mappings, flatDoc, key, v, 0, false); err != nil {
				return nil, err
			}
		}
	}

	var err error
	if mappingsNeedsUpdate {
		if err = index.SetMappings(mappings); err != nil {
			return nil, err
		}
		if err = StoreIndex(index); err != nil {
			return nil, err
		}
	}

	// set timestamp
	timestamp := time.Now()
	if value, ok := flatDoc[meta.TimeFieldName]; ok {
		delete(doc, meta.TimeFieldName)
		prop, _ := mappings.GetProperty(meta.TimeFieldName)
		v, err := zutils.ParseTime(value, prop.Format, prop.TimeZone)
		if err != nil {
			return nil, fmt.Errorf("field [%s] value [%v] parse err: %s", meta.TimeFieldName, value, err.Error())
		}
		timestamp = v
	}

	// prepare for wal
	action := meta.ActionTypeInsert
	if update {
		action = meta.ActionTypeUpdate
	}
	flatDoc[meta.ActionFieldName] = action
	flatDoc[meta.IDFieldName] = docID
	flatDoc[meta.ShardFieldName] = shard
	flatDoc[meta.TimeFieldName] = timestamp.UnixNano()

	return json.Marshal(flatDoc)
}

func (index *Index) checkField(mappings *meta.Mappings, data map[string]interface{}, key string, value interface{}, id int, array bool) error {
	var err error
	var v interface{}
	prop, _ := mappings.GetProperty(key)
	switch prop.Type {
	case "text":
		v, err = zutils.ToString(value)
		if err != nil {
			return fmt.Errorf("field [%s] was set type to [text] but the value [%v] can't convert to string", key, value)
		}
	case "numeric":
		v, err = zutils.ToFloat64(value)
		if err != nil {
			return fmt.Errorf("field [%s] was set type to [numeric] but the value [%v] can't convert to int", key, value)
		}
	case "keyword":
		v, err = zutils.ToString(value)
		if err != nil {
			return fmt.Errorf("field [%s] was set type to [keyword] but the value [%v] can't convert to string", key, value)
		}
	case "bool":
		v, err = zutils.ToBool(value)
		if err != nil {
			return fmt.Errorf("field [%s] was set type to [bool] but the value [%v] can't convert to boolean", key, value)
		}
	case "date", "time":
		_, err := zutils.ParseTime(value, prop.Format, prop.TimeZone)
		if err != nil {
			return fmt.Errorf("field [%s] value [%v] parse err: %s", key, value, err.Error())
		}
		v = value
	}
	if array {
		sub := data[key].([]interface{})
		sub[id] = v
		data[key] = sub
	} else {
		data[key] = v
	}

	return nil
}

// CreateDocument inserts or updates a document in the zinc index
func (index *Index) CreateDocument(docID string, doc map[string]interface{}, update bool) error {
	shard := index.GetShardByDocID(docID)

	// check WAL
	if err := shard.OpenWAL(); err != nil {
		return err
	}

	shardID := shard.GetLatestShardID()
	if update {
		shardID = -1
	}
	data, err := index.CheckDocument(docID, doc, update, shardID)
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
	shardID, err := shard.FindShardByDocID(docID)
	if err != nil {
		if insert && err == errors.ErrorIDNotFound {
			update = false
		} else {
			return err
		}
	}

	data, err := index.CheckDocument(docID, doc, update, shardID)
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

	shardID, err := shard.FindShardByDocID(docID)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		meta.IDFieldName:     docID,
		meta.ActionFieldName: meta.ActionTypeDelete,
		meta.ShardFieldName:  shardID,
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
