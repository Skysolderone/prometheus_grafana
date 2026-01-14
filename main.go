package main

import (
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"

	"v1/prommetrics"
)

func main() {
	// 初始化 Prometheus 指标（只需调用一次）
	prommetrics.Init()

	server := gin.Default()
	// 使用封装好的 Prometheus 中间件
	server.Use(prommetrics.Middleware())
	// 注册 /metrics 路由
	prommetrics.RegisterMetricsRoute(server)

	// 业务接口示例
	server.GET("/get", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello, World!"})
	})

	server.POST("/post", func(c *gin.Context) {
		// 构建随机数，并记录业务指标
		random := rand.Intn(100)
		prommetrics.ObserveRandom(random)

		if random < 50 {
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

	server.Run(":8080")
}
