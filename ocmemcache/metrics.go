package ocmemcache

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// taken from ocgrpc (https://github.com/census-instrumentation/opencensus-go/blob/master/plugin/ocgrpc/stats_common.go)
	latencyDistribution = view.Distribution(0, 0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)

	bytesDistribution = view.Distribution(0, 24, 32, 64, 128, 256, 512, 1024, 2048, 4096, 16384, 65536, 262144, 1048576)
)

var (
	NodeKey, _      = tag.NewKey("node_id")
	OperationKey, _ = tag.NewKey("operation")
)

var (
	CacheOperationSuccessMetric = stats.Int64("mnemosyne/cache_operation_success", "Number of successful cache operations", stats.UnitDimensionless)
	CacheOperationFailureMetric = stats.Int64("mnemosyne/cache_operation_failure", "Number of failed cache operations", stats.UnitDimensionless)
	CacheOperationLatencyMetric = stats.Float64("mnemosyne/cache_operation_latency", "Time taken by the cache operation", stats.UnitMilliseconds)
	ConnectionErrorsMetric      = stats.Int64("mnemosyne/connection_errors", "Connection errors", stats.UnitDimensionless)
	RequestBytesMetric          = stats.Float64("mnemosyne/request_bytes", "Size of request", stats.UnitBytes)
	ResponseBytesMetric         = stats.Float64("mnemosyne/response_bytes", "Size of response", stats.UnitBytes)
)

var (
	CacheOperationSuccessView = &view.View{
		Measure:     CacheOperationSuccessMetric,
		TagKeys:     []tag.Key{OperationKey},
		Aggregation: view.Sum(),
	}

	CacheOperationFailureView = &view.View{
		Measure:     CacheOperationFailureMetric,
		TagKeys:     []tag.Key{OperationKey},
		Aggregation: view.Sum(),
	}

	CacheOperationLatencyView = &view.View{
		Measure:     CacheOperationLatencyMetric,
		TagKeys:     []tag.Key{OperationKey},
		Aggregation: latencyDistribution,
	}

	ConnectionErrorsView = &view.View{
		Measure:     ConnectionErrorsMetric,
		TagKeys:     []tag.Key{NodeKey},
		Aggregation: view.Sum(),
	}

	RequestBytesView = &view.View{
		Measure:     RequestBytesMetric,
		TagKeys:     []tag.Key{OperationKey, NodeKey},
		Aggregation: bytesDistribution,
	}

	ResponseBytesView = &view.View{
		Measure:     ResponseBytesMetric,
		TagKeys:     []tag.Key{OperationKey, NodeKey},
		Aggregation: bytesDistribution,
	}

	DefaultViews = []*view.View{
		CacheOperationSuccessView,
		CacheOperationFailureView,
		CacheOperationLatencyView,
		ConnectionErrorsView,
		RequestBytesView,
		ResponseBytesView,
	}
)
