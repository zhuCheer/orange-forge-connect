# 🍊 Orange Forge Connect SDK

## 🌉 Bridging Servers and Clients with Elegance

> **Secure, Scalable, and Simple Reverse Communication Library for Go**

---

## 🔍 What is Orange Forge Connect?

Orange Forge Connect is a **powerful reverse communication SDK** that enables servers to efficiently send commands to multiple clients without requiring clients to expose any ports. Built with simplicity and security in mind, it's the perfect solution for DevOps, automated operations, and remote command execution scenarios.

---

## ✨ Key Features

- 🔄 **Reverse Communication** - Server can push commands to clients without exposed ports
- 🚀 **Simple Integration** - Minimal code required on both server and client sides
- 🔌 **HTTP Polling** - No complex connection management, easy to debug and maintain
- 🔍 **Transparent Design** - Clear communication flow and straightforward architecture
- 🔐 **Enhanced Security** - Clients initiate all connections, no inbound ports needed
- 🌐 **Distributed Ready** - Stateless server design for horizontal scaling

---

## 🛠️ Perfect For

- 🖥️ **DevOps Automation** - Deploy and manage services across multiple machines
- 🔔 **Health Monitoring** - Collect status reports from distributed clients
- 📦 **Batch Operations** - Execute commands across your entire infrastructure
- 🔧 **Remote Management** - Control and configure remote systems securely

---

## 📊 Why Choose Orange Forge Connect?

| Feature | 🍊 Orange Forge | WebSocket Solutions | Direct HTTP Calls |
|---------|----------------|---------------------|-------------------|
| Security | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| Simplicity | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| Scalability | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| Debugging | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| Implementation | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |

---

## 🚀 Getting Started in Minutes

### 📥 Installation

```bash
go get github.com/zhuCheer/orange-forge-connect
```

### 🖥️ Server Setup

```go
// Initialize Redis connection
conn := redisPool.Get()
defer conn.Close()

// Create and bind the server handler
serverHttpHandler := ForgeServer.WithRdx(conn).Handler()
serverHttpHandler.ServeHTTP(c.Writer, c.Request)
```

### 📱 Client Setup

```go
// Initialize the client
client := forge_connect.NewForge("appid", "secret").
    SetDebug(true).
    SetServerAddr("http://127.0.0.1:8890")

// Register a callback function
client.Regist(func(task *forge_connect.Task) string {
    // Handle the task from server
    return "Task completed successfully!"
})
```

---

## 🔄 How It Works

```
┌─────────┐                                  ┌─────────┐
│         │  1. Poll for tasks periodically  │         │
│ Client  │ ─────────────────────────────>  │ Server  │
│         │                                  │         │
│         │  2. Receive task if available    │         │
│         │ <─────────────────────────────  │         │
│         │                                  │         │
│         │  3. Execute task & return result │         │
│         │ ─────────────────────────────>  │         │
└─────────┘                                  └─────────┘
```

---

## 👨‍💻 Community & Support

- 📝 **Documentation**: Comprehensive guides and API references
- 🐛 **Issue Tracking**: Fast response to bugs and feature requests
- 🤝 **Contributions**: PRs are always welcome!
- 💬 **Community**: Growing network of developers and users

---

## 📜 License

Released under the MIT License - feel free to use, modify, and distribute!

---

## 🔗 Links

- [GitHub Repository](https://github.com/zhuCheer/orange-forge-connect)
- [Examples](https://github.com/zhuCheer/orange-forge-connect/tree/main/example)
- [Report Issues](https://github.com/zhuCheer/orange-forge-connect/issues)

---

**🍊 Orange Forge Connect** - *Connecting your infrastructure, one poll at a time.*
# 