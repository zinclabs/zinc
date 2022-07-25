module github.com/zinclabs/zinc

go 1.16

require (
	github.com/aws/aws-sdk-go-v2/config v1.15.14
	github.com/aws/aws-sdk-go-v2/service/s3 v1.27.1
	github.com/blugelabs/bluge v0.1.9
	github.com/blugelabs/bluge_segment_api v0.2.0
	github.com/blugelabs/ice v1.0.0
	github.com/blugelabs/query_string v0.3.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bwmarrin/snowflake v0.3.0
	github.com/dgraph-io/badger/v3 v3.2103.2
	github.com/getsentry/sentry-go v0.13.0
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-contrib/pprof v1.4.0
	github.com/gin-gonic/gin v1.8.1
	github.com/go-ego/gse v0.70.2
	github.com/goccy/go-json v0.9.10
	github.com/joho/godotenv v1.4.0
	github.com/minio/minio-go/v7 v7.0.31
	github.com/pyroscope-io/client v0.3.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.27.0
	github.com/segmentio/analytics-go/v3 v3.2.1
	github.com/shirou/gopsutil/v3 v3.22.6
	github.com/stretchr/testify v1.8.0
	github.com/swaggo/files v0.0.0-20220610200504-28940afbdbfe
	github.com/swaggo/gin-swagger v1.5.1
	github.com/swaggo/swag v1.8.3
	github.com/zinclabs/wal v1.2.2
	github.com/zsais/go-gin-prometheus v0.1.0
	go.etcd.io/bbolt v1.3.6
	go.etcd.io/etcd/client/v3 v3.5.4
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f
	golang.org/x/text v0.3.7

)

replace github.com/blugelabs/bluge => github.com/zinclabs/bluge v1.1.6-rc-6

replace github.com/blugelabs/ice => github.com/zinclabs/ice v1.1.3

replace github.com/blugelabs/bluge_segment_api => github.com/zinclabs/bluge_segment_api v1.0.0
