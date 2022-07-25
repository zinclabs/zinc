package index

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zinclabs/zinc/pkg/core"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/meta/elastic"
)

// @Summary Get index mappings for compatible ES
// @Tags    Index
// @Produce json
// @Param   target path  string  true  "Index"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} meta.HTTPResponse
// @Router /es/:target/_mapping [get]
func GetESMapping(c *gin.Context) {
	indexName := c.Param("target")
	index, exists := core.GetIndex(indexName)
	if !exists {
		c.JSON(http.StatusBadRequest, meta.HTTPResponseError{Error: "index " + indexName + " does not exists"})
		return
	}

	// format mappings
	mappings := index.GetMappings()

	// NOTE: Zinc currently "converts" object array fields to "field.index.sub_field"
	// Example Input Document:
	//  {
	//    "field": [
	//      {
	//        "sub_field": true
	//      },
	//      {
	//        "sub_field": false
	//      },
	//    ]
	//  }
	//
	// The resulting index mappings will be:
	//   * field.0.sub_field
	//   * field.1.sub_field
	// Which is not compatible with ES – to provide the best compatibility, the index number will be
	// kept in the resulting mapping.
	es := convertToESMapping(mappings)

	c.JSON(http.StatusOK, gin.H{index.GetName(): gin.H{"mappings": es}})
}

// convertToESMapping converts the given Zinc mappings to the ElasticSearch representation.
// TODO: In the future, the result can be stored, if a performance gain can be achieved (with a TTL).
func convertToESMapping(mappings *meta.Mappings) *elastic.Mappings {
	orig := mappings.DeepClone()
	m := elastic.NewMappings()

	// we first have to remove the automatically added property field mappings
	for k, v := range orig.Properties {
		if v.Fields != nil {
			for fKey := range v.Fields {
				delete(orig.Properties, k+"."+fKey)
			}
		}
	}

	// ES returns the mapping unflattened
	keys := make([]string, 0, len(orig.Properties))
	for k := range orig.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, idx := range keys {
		// we skip any top-level property
		if !strings.Contains(idx, ".") {
			if origProp, exists := orig.GetProperty(idx); exists {
				m.SetProperty(idx, convertToESProperty(origProp))
			}

			continue
		}

		// TODO: What is with fields which contain explicitly periods, e.g. "hello.world"?
		strs := strings.Split(idx, ".")
		tmp := strs[0]

		p := elastic.NewProperty("")
		if origProp, exists := orig.GetProperty(tmp); exists {
			p = convertToESProperty(origProp)
		} else if prop, exists := m.GetProperty(tmp); exists {
			p = prop
		}

		var field elastic.Property
		for i, str := range strs[1:] {
			tmp += "." + str

			subField := elastic.NewProperty("")
			if sub, exist := orig.GetProperty(tmp); exist {
				subField = convertToESProperty(sub)
			} else {
				t := field
				if i == 0 {
					t = p
				}

				if prop, exist := t.Properties[str]; exist {
					subField = prop
				}
			}

			if i == 0 {
				p.Properties[str] = subField
			} else {
				field.Properties[str] = subField
			}

			field = subField
		}

		m.SetProperty(strs[0], p)
	}

	return m
}

// convertToESProperty converst the given property to the ES representation.
func convertToESProperty(p meta.Property) elastic.Property {
	p = p.DeepClone()

	prop := elastic.NewProperty(p.Type)
	prop.Analyzer = p.Analyzer
	prop.SearchAnalyzer = p.SearchAnalyzer
	prop.Format = p.Format

	if p.Fields != nil {
		for k, v := range p.Fields {
			prop.Fields[k] = convertToESProperty(v)
		}
	}

	return prop
}
