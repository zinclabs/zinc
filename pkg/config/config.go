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

package config

import (
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/blugelabs/ice/compress"
)

type config struct {
	GinMode                   string `env:"GIN_MODE"`
	ServerPort                string `env:"ZINC_SERVER_PORT,default=4080"`
	ServerMode                string `env:"ZINC_SERVER_MODE,default=node"`
	NodeID                    string `env:"ZINC_NODE_ID,default=1"`
	DataPath                  string `env:"ZINC_DATA_PATH,default=./data"`
	MetadataStorage           string `env:"ZINC_METADATA_STORAGE,default=bolt"`
	IceCompressor             string `env:"ZINC_ICE_COMPRESSOR,default=zstd"`
	SentryEnable              bool   `env:"ZINC_SENTRY,default=true"`
	SentryDSN                 string `env:"ZINC_SENTRY_DSN,default=https://15b6d9b8be824b44896f32b0234c32b7@o1218932.ingest.sentry.io/6360942"`
	ProfilerEnable            bool   `env:"ZINC_PROFILER,default=false"`
	ProfilerServer            string `env:"ZINC_PROFILER_SERVER,default=https://pyroscope.dev.zincsearch.com"`
	ProfilerAPIKey            string `env:"ZINC_PROFILER_API_KEY,default=psx-AfPbC5Bh6gI4dHkCMpoxM2Qd7Xblsqhip5nlwvHdhAE1"`
	ProfilerFriendlyProfileID string `env:"ZINC_PROFILER_FRIENDLY_PROFILE_ID"`
	TelemetryEnable           bool   `env:"ZINC_TELEMETRY,default=true"`
	PrometheusEnable          bool   `env:"ZINC_PROMETHEUS_ENABLE,default=false"`
	EnableTextKeywordMapping  bool   `env:"ZINC_ENABLE_TEXT_KEYWORD_MAPPING,default=false"`
	BatchSize                 int    `env:"ZINC_BATCH_SIZE,default=1024"`
	MaxResults                int    `env:"ZINC_MAX_RESULTS,default=10000"`
	AggregationTermsSize      int    `env:"ZINC_AGGREGATION_TERMS_SIZE,default=1000"`
	WalSyncInterval           string `env:"ZINC_WAL_SYNC_INTERVAL,default=1s"`      // sync wal to disk, 1s, 10ms
	WalRedoLogNoSync          bool   `env:"ZINC_WAL_REDOLOG_NO_SYNC,default=false"` // control sync after every write
	ReadGorutineNum           int    `env:"ZINC_READ_GORUTINE_NUM,default=10"`      // control gorutine number for read
	Shard                     shard
	Etcd                      etcd
	S3                        s3
	MinIO                     minIO
	Plugin                    plugin
}

// 1073741824 1g
// 536870912  512m
// 268435456  256m
// 134217728  128m
// 67108864   64m
// 33554432   32m
// 16777216   16m

type shard struct {
	// DefaultNum is the default number of shards.
	Num int64 `env:"ZINC_SHARD_NUM,default=5"`
	// MaxSize is the maximum size limit for one shard, or will create a new shard.
	MaxSize uint64 `env:"ZINC_SHARD_MAX_SIZE,default=1073741824"`
}

type etcd struct {
	Endpoints []string `env:"ZINC_ETCD_ENDPOINTS"`
	Prefix    string   `env:"ZINC_ETCD_PREFIX,default=/zinc"`
	Username  string   `env:"ZINC_ETCD_USERNAME"`
	Password  string   `env:"ZINC_ETCD_PASSWORD"`
}

type s3 struct {
	Bucket string `env:"ZINC_S3_BUCKET"`
}

type minIO struct {
	Endpoint        string `env:"ZINC_MINIO_ENDPOINT"`
	Bucket          string `env:"ZINC_MINIO_BUCKET"`
	AccessKeyID     string `env:"ZINC_MINIO_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"ZINC_MINIO_SECRET_ACCESS_KEY"`
}

type plugin struct {
	ES  elasticsearch
	GSE gse
}

type elasticsearch struct {
	Version string `env:"ZINC_PLUGIN_ES_VERSION"`
}

type gse struct {
	Enable    bool   `env:"ZINC_PLUGIN_GSE_ENABLE,default=false"`
	DictEmbed string `env:"ZINC_PLUGIN_GSE_DICT_EMBED,default=small"`
	DictPath  string `env:"ZINC_PLUGIN_GSE_DICT_PATH,default=./plugins/gse/dict"`
}

var Global = new(config)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Print(err.Error())
	}
	rv := reflect.ValueOf(Global).Elem()
	loadConfig(rv)

	// configure gin
	if Global.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// check data path
	testPath := path.Join(Global.DataPath, "_test_")
	if err := os.MkdirAll(testPath, 0755); err != nil {
		log.Fatal().Err(err).Msg("ZINC_DATA_PATH is not writable")
	}
	if err := os.Remove(testPath); err != nil {
		log.Fatal().Err(err).Msg("ZINC_DATA_PATH is not writable")
	}

	// configure ice compress algorithm
	switch strings.ToUpper(Global.IceCompressor) {
	case "SNAPPY":
		compress.Algorithm = compress.SNAPPY
	case "S2":
		compress.Algorithm = compress.S2
	case "ZSTD":
		compress.Algorithm = compress.ZSTD
	}
}

func loadConfig(rv reflect.Value) {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if ft.Type.Kind() == reflect.Struct {
			loadConfig(fv)
			continue
		}
		if ft.Tag.Get("env") != "" {
			tag := ft.Tag.Get("env")
			setField(fv, tag)
		}
	}
}

func setField(field reflect.Value, tag string) {
	if tag == "" {
		return
	}
	tagColumn := strings.Split(tag, ",")
	v := os.Getenv(tagColumn[0])
	if v == "" {
		if len(tagColumn) > 1 {
			tv := strings.Join(tagColumn[1:], ",")
			if strings.HasPrefix(tv, "default=") {
				v = tv[8:]
			}
		}
	}
	if v == "" {
		return
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vi, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Fatal().Err(err).Msgf("env %s is not int", tag)
		}
		field.SetInt(int64(vi))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vi, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			log.Fatal().Err(err).Msgf("env %s is not uint", tag)
		}
		field.SetUint(uint64(vi))
	case reflect.Bool:
		vi, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatal().Err(err).Msgf("env %s is not bool", tag)
		}
		field.SetBool(vi)
	case reflect.String:
		field.SetString(v)
	case reflect.Slice:
		vs := strings.Split(v, ",")
		field.Set(reflect.ValueOf(vs))
		field.SetLen(len(vs))
	}
}
