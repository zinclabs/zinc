package test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestApiES(t *testing.T) {
	Convey("test es api", t, func() {
		Convey("POST /es/_bulk", func() {
			Convey("bulk documents", func() {
				body := bytes.NewBuffer(nil)
				body.WriteString(bulkData)
				resp := request("POST", "/es/_bulk", body)
				So(resp.Code, ShouldEqual, http.StatusOK)
			})
			Convey("bulk documents with delete", func() {
				body := bytes.NewBuffer(nil)
				body.WriteString(bulkDataWithDelete)
				resp := request("POST", "/es/_bulk", body)
				So(resp.Code, ShouldEqual, http.StatusOK)
			})
			Convey("bulk with error input", func() {
				body := bytes.NewBuffer(nil)
				body.WriteString(`{"index":{}}`)
				resp := request("POST", "/es/_bulk", body)
				So(resp.Code, ShouldEqual, http.StatusOK)
			})
		})

		Convey("POST /es/:target/_bulk", func() {
			Convey("bulk create documents with not exist indexName", func() {
				body := bytes.NewBuffer(nil)
				data := strings.ReplaceAll(bulkData, `"_index": "games3"`, `"_index": ""`)
				body.WriteString(data)
				resp := request("POST", "/es/notExistIndex/_bulk", body)
				So(resp.Code, ShouldEqual, http.StatusOK)
			})
			Convey("bulk create documents with exist indexName", func() {
				// create index
				body := bytes.NewBuffer(nil)
				body.WriteString(`{"name": "` + indexName + `", "storage_type": "disk"}`)
				resp := request("PUT", "/api/index", body)
				So(resp.Body.String(), ShouldEqual, "")
				So(resp.Code, ShouldEqual, http.StatusOK)

				// check bulk
				body.Reset()
				data := strings.ReplaceAll(bulkData, `"_index": "games3"`, `"_index": ""`)
				body.WriteString(data)
				resp = request("POST", "/es/"+indexName+"/_bulk", body)
				So(resp.Code, ShouldEqual, http.StatusOK)
			})
			Convey("bulk with error input", func() {
				body := bytes.NewBuffer(nil)
				body.WriteString(`{"index":{}}`)
				resp := request("POST", "/es/_bulk", body)
				So(resp.Code, ShouldEqual, http.StatusOK)
			})
		})

		Convey("POST /es/:target/_doc", func() {
			Convey("create document with not exist indexName", func() {
			})
			Convey("create document with exist indexName", func() {
			})
			Convey("create document with exist indexName not exist id", func() {
			})
			Convey("create document with exist indexName and exist id", func() {
			})
			Convey("create document with error input", func() {
			})
		})

		Convey("PUT /es/:target/_doc/:id", func() {
			Convey("create document with not exist indexName", func() {
			})
			Convey("create document with exist indexName", func() {
			})
			Convey("create document with exist indexName not exist id", func() {
			})
			Convey("create document with exist indexName and exist id", func() {
			})
			Convey("create document with error input", func() {
			})
		})

		Convey("DELETE /es/:target/_doc/:id", func() {
			Convey("delete document with not exist indexName", func() {
			})
			Convey("delete document with exist indexName", func() {
			})
			Convey("delete document with exist indexName not exist id", func() {
			})
			Convey("delete document with exist indexName and exist id", func() {
			})
			Convey("delete document with error input", func() {
			})
		})

		Convey("PUT /es/:target/_create/:id", func() {
			Convey("create document with not exist indexName", func() {
			})
			Convey("create document with exist indexName", func() {
			})
			Convey("create document with exist indexName not exist id", func() {
			})
			Convey("create document with exist indexName and exist id", func() {
			})
			Convey("create document with error input", func() {
			})
		})

		Convey("POST /es/:target/_create/:id", func() {
			Convey("create document with not exist indexName", func() {
			})
			Convey("create document with exist indexName", func() {
			})
			Convey("create document with exist indexName not exist id", func() {
			})
			Convey("create document with exist indexName and exist id", func() {
			})
			Convey("create document with error input", func() {
			})
		})

		Convey("POST /es/:target/_update/:id", func() {
			Convey("update document with not exist indexName", func() {
			})
			Convey("update document with exist indexName", func() {
			})
			Convey("update document with exist indexName not exist id", func() {
			})
			Convey("update document with exist indexName and exist id", func() {
			})
			Convey("update document with error input", func() {
			})
		})

	})
}
