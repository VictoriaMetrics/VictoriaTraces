module github.com/VictoriaMetrics/VictoriaTraces

go 1.24.4

replace github.com/VictoriaMetrics/VictoriaMetrics => github.com/VictoriaMetrics/VictoriaMetrics v0.0.0-20250706151707-8646b73efc0e

require (
	github.com/VictoriaMetrics/VictoriaLogs v1.25.0
	github.com/VictoriaMetrics/VictoriaMetrics v0.0.0-00010101000000-000000000000
	github.com/VictoriaMetrics/easyproto v0.1.4
	github.com/VictoriaMetrics/metrics v1.37.0
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/google/go-cmp v0.7.0
	github.com/valyala/fastjson v1.6.4
	github.com/valyala/fastrand v1.1.0
	github.com/valyala/quicktemplate v1.8.0
	golang.org/x/time v0.12.0
)

require (
	github.com/VictoriaMetrics/metricsql v0.84.6 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/valyala/gozstd v1.22.0 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)
