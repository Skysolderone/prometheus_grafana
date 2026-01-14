package main

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 不统计 Prometheus 自己拉取指标的请求，避免 /metrics 污染业务指标
		if c.FullPath() == "/metrics" {
			c.Next()
			return
		}

		// 记录请求大小
		requestSize := float64(c.Request.ContentLength)
		if requestSize < 0 {
			requestSize = 0
		}

		// 记录开始时间和并发请求数
		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// 包装 ResponseWriter 以捕获响应大小
		wrapped := &responseWriter{
			ResponseWriter: c.Writer,
			statusCode:     200,
			bytesWritten:   0,
		}
		c.Writer = wrapped

		c.Next()

		// 计算持续时间
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(wrapped.statusCode)
		method := c.Request.Method
		path := c.FullPath()

		// 记录所有指标
		httpRequests.WithLabelValues(method, path, statusCode).Inc()
		httpDuration.WithLabelValues(method, path).Observe(duration)
		httpRequestDuration.WithLabelValues(method, path, statusCode).Observe(duration)
		httpRequestSize.WithLabelValues(method, path).Observe(requestSize)
		httpResponseSize.WithLabelValues(method, path, statusCode).Observe(float64(wrapped.bytesWritten))

		// 记录错误指标
		if wrapped.statusCode >= 400 {
			httpErrors.WithLabelValues(method, path, statusCode).Inc()
		}

		// 记录成功指标
		if wrapped.statusCode >= 200 && wrapped.statusCode < 300 {
			httpSuccess.WithLabelValues(method, path).Inc()
		}

		// 记录超时指标（如果超过阈值）
		if duration > 1.0 {
			httpSlowRequests.WithLabelValues(method, path).Inc()
		}
	}
}

// responseWriter 用于捕获响应大小和状态码
type responseWriter struct {
	gin.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

func (w *responseWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	w.bytesWritten += int64(n)
	return n, err
}

var (
	// HTTP 请求总数（Counter）
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 请求持续时间 - Summary（保留原有）
	httpDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_duration_seconds",
			Help: "HTTP request duration in seconds",
		},
		[]string{"method", "path"},
	)

	// HTTP 请求持续时间 - Histogram（更常用，支持分位数查询）
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 请求大小（Histogram）
	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000, 500000, 1000000},
		},
		[]string{"method", "path"},
	)

	// HTTP 响应大小（Histogram）
	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000, 500000, 1000000},
		},
		[]string{"method", "path", "status"},
	)

	// 当前正在处理的 HTTP 请求数（Gauge）
	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// HTTP 错误请求数（Counter）
	httpErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP error requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 成功请求数（Counter）
	httpSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_success_total",
			Help: "Total number of successful HTTP requests",
		},
		[]string{"method", "path"},
	)

	// HTTP 慢请求数（Counter）
	httpSlowRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_slow_requests_total",
			Help: "Total number of slow HTTP requests (duration > 1s)",
		},
		[]string{"method", "path"},
	)

	// HTTP 请求速率（Gauge，每秒请求数）
	httpRequestRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_request_rate",
			Help: "HTTP request rate per second",
		},
		[]string{"method", "path"},
	)

	// 应用启动时间（Gauge）
	appStartTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_start_time_seconds",
			Help: "Application start time in seconds since epoch",
		},
	)

	// 应用运行时间（Gauge）
	appUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	// 业务指标：随机数生成总数（Counter）
	randomNumberGenerated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "random_number_generated_total",
			Help: "Total number of random numbers generated",
		},
	)

	// 业务指标：随机数生成失败数（Counter）
	randomNumberFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "random_number_failed_total",
			Help: "Total number of failed random number generations",
		},
	)

	// 业务指标：随机数分布（Histogram）
	randomNumberValue = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "random_number_value",
			Help:    "Distribution of generated random number values",
			Buckets: []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	)
)

var (
	// 创建自定义的 Prometheus Registry
	reg = prometheus.NewRegistry()
)

func init() {
	// 注册 Prometheus 默认收集器（Go runtime 指标和进程指标）
	reg.MustRegister(collectors.NewGoCollector())                                       // go_* 指标
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})) // process_* 指标

	// 注册所有自定义指标
	reg.MustRegister(
		httpRequests,
		httpDuration,
		httpRequestDuration,
		httpRequestSize,
		httpResponseSize,
		httpRequestsInFlight,
		httpErrors,
		httpSuccess,
		httpSlowRequests,
		httpRequestRate,
		appStartTime,
		appUptime,
		randomNumberGenerated,
		randomNumberFailed,
		randomNumberValue,
	)

	// 设置应用启动时间
	appStartTime.SetToCurrentTime()
}

func main() {
	server := gin.Default()
	server.Use(PrometheusMiddleware())
	server.GET("/metrics", gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
	server.GET("/get", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello, World!"})
	})
	server.POST("/post", func(c *gin.Context) {
		//构建随机数
		random := rand.Intn(100)
		randomNumberGenerated.Inc()
		randomNumberValue.Observe(float64(random))

		if random < 50 {
			randomNumberFailed.Inc()
			c.JSON(500, gin.H{"message": "Error"})
		} else {
			c.JSON(200, gin.H{"message": "Success", "random": random})
		}
	})

	// 慢接口：用于测试并发效果（模拟处理耗时）
	server.GET("/slow", func(c *gin.Context) {
		// 随机延迟 100-500ms，模拟业务处理时间
		delay := time.Duration(100+rand.Intn(400)) * time.Millisecond
		time.Sleep(delay)
		c.JSON(200, gin.H{"message": "Slow response", "delay_ms": delay.Milliseconds()})
	})

	// 启动后台任务更新运行时间
	startTime := time.Now()
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			appUptime.Set(time.Since(startTime).Seconds())
		}
	}()

	server.Run(":8080")
}
