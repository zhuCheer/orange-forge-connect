package main

import (
	"github.com/gomodule/redigo/redis"
	forge_connect "github.com/zhuCheer/orange-forge-connect"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var ForgeServer *forge_connect.Server
var redisPool *redis.Pool

func main() {
	InitRedisPool()
	InitForge()

	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	// test set up a simple task push to client
	r.GET("/ping", func(c *gin.Context) {
		conn := redisPool.Get()
		defer conn.Close() // Must be closed after use, otherwise the connection will not be returned to the pool

		cc := ForgeServer.WithRdx(conn).WithSingleTimeout(10 * time.Second)

		// 向 appid=abc的客户端发送一个任务，任务类型为PING，消息体为this message is from server
		// respBody is the response body from the client
		_, respBody, _ := cc.RunSingleTask("orange-forge", "PING", "this message is from server")

		c.JSON(http.StatusOK, gin.H{
			"message": "message from client:" + string(respBody),
		})
	})

	// Bind the forge server to gin context
	r.POST("/orange-forge/*any", BindForgeServer())

	// 启动服务
	r.Run(":8003") // 默认监听在 0.0.0.0:8082
}

func InitForge() {
	// Initialize the server object
	ForgeServer = forge_connect.NewServer("orange-forge-board").
		SetDebug().
		SetTaskWaitTick(500 * time.Millisecond)

	// Initialize routes
	ForgeServer.Handler()
}

// InitRedisPool initializes the Redis connection pool
func InitRedisPool() {
	redisPool = &redis.Pool{
		MaxIdle:     1,
		MaxActive:   100, // 0 = unlimited
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "127.0.0.1:6379",
				redis.DialPassword("your-redis-password"),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// BindForgeServer is a gin middleware to bind the forge server to gin context
func BindForgeServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		conn := redisPool.Get()
		defer conn.Close() // Must be closed after use, otherwise the connection will not be returned to the pool
		serverHttpHandler := ForgeServer.WithRdx(conn).Handler()

		// Convert gin context to standard http request and response
		serverHttpHandler.ServeHTTP(c.Writer, c.Request)
	}
}
