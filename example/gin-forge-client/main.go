package main

import (
	"fmt"
	forge_connect "github.com/zhuCheer/orange-forge-connect"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

var ForgeClient *forge_connect.Client

func main() {

	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	// 启动 Forge 客户端
	InitForge()

	// 启动服务
	r.Run(":8004") // 默认监听在 0.0.0.0:8082
}

func InitForge() {
	ForgeClient = forge_connect.NewForge("orange-forge", "123456").
		SetDebug(true).SetServerAddr("http://127.0.0.1:8003")

	_, _, err := ForgeClient.Regist(CallbackTask)
	if err != nil {
		// Error handling
		log.Fatalf(err.Error())
	}

}

// CallbackTask is the callback function that will be called by Forge
func CallbackTask(task *forge_connect.Task) (result string) {
	result = "this message from forge client now time is " + time.Now().String()
	fmt.Println(result)
	return
}
