# orange-forge-agent 反向通信库

## 背景介绍

在传统的 B/S 架构中，服务端通常通过 HTTP 服务与浏览器、CURL、Python 等客户端进行通信，用户可以直接通过浏览器、命令行工具或编程语言的 HTTP 库（如 Python 的 requests）访问服务端接口。

### 传统 B/S 架构通信流程

```mermaid
sequenceDiagram
    participant Client as 浏览器/脚本/工具
    participant Server as 服务端
    Client->>Server: HTTP 请求（如注册、查询、操作）
    Server-->>Client: 返回响应（数据/状态）
```

然而，在 DevOps、自动化运维等场景下，常常需要服务端主动下发指令到大量客户端（如批量部署、健康检查等）。如果让每台目标机器都暴露 HTTP 服务供平台调用，既不安全也不现实。因此，我们需要一种能够从服务端高效、安全地下发指令到客户端集群的通信组件。

### 传统服务端主动下发指令的局限性

```mermaid
flowchart TD
    Server[服务端]
    Client1[客户端A暴露HTTP服务]
    Client2[客户端B暴露HTTP服务]
    ClientN[客户端N暴露HTTP服务]
    Server -- 访问 --> Client1
    Server -- 访问 --> Client2
    Server -- 访问 --> ClientN
```

> 每台客户端都需暴露端口，安全性和可维护性差

### 典型场景
- DevOps 工具中的批量服务部署
- 客户端健康检查与状态上报
- 自动化运维批量任务下发
- 远程命令执行

## 通信方式对比

为实现服务端与客户端的双向通信，常见有两种方案：

1. **长连接（如 WebSocket）**
   - 优点：通信实时、流畅，信息传递延迟低。
   - 缺点：分布式部署需引入连接管理组件，开发和运维复杂，异常排查困难。

2. **HTTP 轮询**
   - 优点：实现简单，服务端易于分布式扩展，无需复杂组件，排查问题容易。
   - 缺点：通信实时性依赖轮询频率，体验略有顿挫。

本库采用第二种方案——**HTTP 轮询**，对服务端和客户端进行了高度封装，仅需简单调用即可实现高效的反向通信服务。

---

## 特性

- **HTTP轮询反向通信**：无需暴露客户端端口，服务端可主动下发指令
- **分布式友好**：服务端无状态，易于横向扩展
- **高度封装**：服务端、客户端均只需少量代码即可集成
- **易于排查**：无复杂连接管理，问题定位简单
- **适用广泛**：批量任务、健康检查、远程命令等

---

## 安装

```bash
go get github.com/zhuCheer/orange-forge-connect
```

---

## 快速开始

### 服务端示例

```go
// 初始化服务端对象
service.ForgeServer = forge_connect.NewServer("orange-forge-board").
    SetDebug().
    SetTaskWaitTick(500 * time.Millisecond)

// 初始化路由
service.ForgeServer.Handler()
```

```go
// 路由绑定
r.ALL("/orange-forge/*", controller.ServerApi)

func ServerApi(c *app.Context) error {
    rdxconn, err := c.GetRedis("default")
    if err != nil {
        define.ErrorData(c, 500, err.Error())
    }
    serverHttpHandler := service.ForgeServer.WithRdx(rdxconn).Handler()
    c.RunHttpHandler(serverHttpHandler)
    return nil
}
```

### 客户端示例

```go
// 初始化客户端
service.ForgeClient = forge_connect.NewForge("appid", "secret").
    SetDebug(true).
    SetServerAddr("http://127.0.0.1:8890")

// 注册回调方法
_, _, err := service.ForgeClient.Regist(CallbackTask)
if err != nil {
    // 错误处理
}

// 回调方法示例
func CallbackTask(task *forge_connect.Task) (result string) {
    logger.Infow("CallbackTask Run------------->", "task", task)
    if task == nil {
        return "not found task"
    }
    return "0000000000000000000"
}
```

---

## 通信流程

### 1. 客户端主动调用服务端

```mermaid
sequenceDiagram
    participant Client
    participant Server
    Client->>Server: 注册/心跳/拉取任务（HTTP）
    Server-->>Client: 返回响应（任务/状态）
```

### 2. 服务端主动下发指令到客户端（轮询模式）

```mermaid
flowchart TD
    subgraph 轮询循环
        Client-->|定时轮询|Server
        Server-->|有任务|Client
        Client-->|回传结果|Server
        Server-->|无任务|Client
    end
```

---

## 典型应用场景

- DevOps 自动化运维
- 批量任务下发
- 客户端健康检查
- 远程命令执行

---

## 贡献

欢迎 Issue 和 PR！如有建议或需求，欢迎提交。

---

## License

MIT

---

如需更多细节，请参考源码及注释。

