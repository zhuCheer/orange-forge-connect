package forge_connect

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type TaskFunc func(task *Task) (result string)

type Client struct {
	AppID         string
	IsDebug       bool
	registered    bool
	secret        string
	baseSecret    string
	serverAddr    string
	mu            sync.Mutex
	checkInterval int
	taskInterval  time.Duration
	skipSSL       bool
	HttpClient    *http.Client
	callbackFunc  TaskFunc
}

// NewForge initializes the client configuration
func NewForge(appID, secret string) *Client {
	if appID == "" || secret == "" {
		panic("appid/secret not found.")
	}
	return &Client{
		AppID:         appID,
		secret:        secret,
		baseSecret:    DEFAULT_SECRET,
		checkInterval: 10,
		taskInterval:  1 * time.Second,
		HttpClient:    &http.Client{Timeout: 60 * time.Second},
	}
}

// SetHttpClient updates the client configuration of http client
func (c *Client) SetHttpClient(client *http.Client) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HttpClient = client
	return c
}

// SetDebug set debug model
func (c *Client) SetDebug(debug bool) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsDebug = debug
	return c
}

// GetServerAddr get server addr
func (c *Client) GetServerAddr() (addr string) {
	return c.serverAddr
}

// GetConnecteState returns the connection state of the client
func (c *Client) GetConnecteState() (connected bool) {
	return c.registered
}

// SetServerAddr updates the client configuration of server addr
func (c *Client) SetServerAddr(addr string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if addr != "" {
		c.serverAddr = addr
	}
	return c
}

// SetSkipSSL  Skip ssl validity verification
func (c *Client) SetSkipSSL(i bool) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skipSSL = i
	return c
}

func (c *Client) SetTaskDelay(timeout time.Duration) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if timeout > 0 {
		c.taskInterval = timeout
	}
	return c
}

// generateSignature creates an HMAC-SHA256 signature based on appName, secret, and the provided payload.
func (c *Client) generateSignature(api, dateTime, payload string) string {
	secret := c.secret
	if api == "register" {
		secret = c.baseSecret
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(c.AppID + payload + dateTime))
	return hex.EncodeToString(h.Sum(nil))
}

// generateSignatureBySecret creates an HMAC-SHA256 signature based on appName, secret, and the provided payload.
func (c *Client) generateSignatureBySecret(secret, dateTime, payload string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(c.AppID + payload + dateTime))
	return hex.EncodeToString(h.Sum(nil))
}

// ensureServerAddr checks if serverAddr is set
func (c *Client) ensureConfig() {
	if c.serverAddr == "" {
		panic("serverAddr is not set")
	}

	_, err := url.ParseRequestURI(c.serverAddr)
	if err != nil {
		panic("serverAddr is invalid:" + err.Error())
	}

}

// Regist regist app info for server
func (c *Client) Regist(callback TaskFunc) (respData string, errno int, err error) {
	c.ensureConfig()
	params := RegistrationRequest{
		AppID:  c.AppID,
		Secret: c.secret,
	}
	paramsJson, _ := json.Marshal(params)

	resp, errno, err := c.SendHTTPRequest("register", string(paramsJson))

	if err != nil {
		return
	}
	respData, _ = resp.(string)
	if callback != nil {
		c.callbackFunc = callback
	}
	c.registered = true
	consoleLog("INFO", "AgentInit success <===> forgeServer %s", c.serverAddr)
	go c.helthCheck(c.checkInterval)
	go c.listenGetTask()
	return
}

// Start starts the client to listen for tasks and perform health checks
func (c *Client) listenGetTask() {
	isRegistStatus := c.GetConnecteState()
	if isRegistStatus == false {
		return
	}
	if c.callbackFunc == nil {
		consoleLog("ERROR", "callbackFunc is not set, please set it before starting the client.")
		return
	}

	for {
		task, errno, err := c.GetTask()
		if err != nil && errno == 2 {
			if c.IsDebug {
				consoleLog("DEBUG", "GetTask context canceled or no task.")
			}
		}
		if err != nil && errno != 2 {
			consoleLog("ERROR", "GetTask context error: %v, errno: %d", err, errno)
			break
		}
		if task != nil {
			go func() {
				result := c.callbackFunc(task)
				task.Result = result
				task.DoStatus = STATUS_SUCCESS
				c.pushTaskResult(task)
			}()
		}
		time.Sleep(c.taskInterval)
	}
	consoleLog("ERROR", "------------disconnect server------------")
	os.Exit(1)
}

// Ping sends a ping request to the server
func (c *Client) pushTaskResult(task *Task) {
	c.ensureConfig()
	params, _ := json.Marshal(task)
	resp, _, err := c.SendHTTPRequest("reportTask", string(params))

	if err != nil {
		consoleLog("ERROR", "pushTaskResult response: %v", resp)
		return
	}
	respData, _ := resp.(string)

	consoleLog("DEBUG", "pushTaskResult response: %s", respData)
	return
}

// helthCheck 连接健康检查
func (c *Client) helthCheck(second int) {
	isRegistStatus := c.GetConnecteState()
	if isRegistStatus == false {
		return
	}
	// Close health check before exiting the process
	// 创建 Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interval := second
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	errCnt := 0
	for {
		select {
		case <-ctx.Done():
			consoleLog("INFO", "ConnectHealthCheck ticker stopped.")
			return
		case <-ticker.C:
			if c.IsDebug {
				consoleLog("DEBUG", "ConnectHealthCheck interval %v.", second)
			}

			err := c.Ping()
			if err != nil {
				errCnt++
				interval = fibonacciBackoff(errCnt, 7200)
				if errCnt > 3 {
					consoleLog("ERROR", "ConnectHealthCheck err %v, errCnt:%v, interval:%v", err.Error(), errCnt, interval)
					c.registered = false
				}
			} else {
				errCnt = 0
				interval = second
			}

			ticker.Stop()
			ticker = time.NewTicker(time.Duration(interval) * time.Second)
		}
	}
}

// Ping sends a ping request to the server
func (c *Client) Ping() (err error) {
	c.ensureConfig()
	resp, _, err := c.SendHTTPRequest("ping", "ping")

	if err != nil {
		return err
	}
	respData, _ := resp.(string)

	if c.IsDebug {
		consoleLog("DEBUG", "ping response: %s", respData)
	}

	return
}

// GetTask polls the server for a new task
func (c *Client) GetTask() (task *Task, errno int, err error) {
	c.ensureConfig()
	resp, errno, err := c.SendHTTPRequest("getTask", "")
	if err != nil {
		return nil, errno, err
	}

	respJson, _ := json.Marshal(resp)
	respData := Task{}
	json.Unmarshal(respJson, &respData)

	return &respData, 0, nil
}

// SendHTTPRequest 发送HTTP请求并返回响应结果
func (c *Client) SendHTTPRequest(api, payload string) (interface{}, int, error) {
	var apiResp interface{}
	// 创建HTTP客户端并设置超时时间
	client := c.HttpClient
	apiUrl := c.serverAddr + c.getApi(api)

	req, err := http.NewRequest("POST", apiUrl, bytes.NewReader([]byte(payload)))
	if err != nil {
		return apiResp, 1, fmt.Errorf("failed to create request: %v", err)
	}

	dateTime := TimeFormat(time.Now())
	signature := c.generateSignature(api, dateTime, payload)

	reqHeader := map[string]string{
		"X-FORGE-SIGN":  signature,
		"X-FORGE-APPID": c.AppID,
		"X-FORGE-TIME":  dateTime,
		"Content-Type":  "application/json",
	}
	for key, value := range reqHeader {
		req.Header.Set(key, value)
	}
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", 1, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return apiResp, 1, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, payload)
	}
	respData := Response{}
	err = json.Unmarshal(body, &respData)
	if err != nil {
		return apiResp, 1, err
	}

	if respData.Code > 0 {
		return apiResp, respData.Code, errors.New(respData.Message)
	}
	return respData.Data, 1, nil
}

func (c *Client) getApi(key string) (apiUrl string) {

	apiUrl, _ = apiRoutes[key]

	return
}

// fibonacciBackoff Fibonacci interval calculation (limit the maximum interval time)
func fibonacciBackoff(n int, maxInterval int) int {
	if n <= 1 {
		return 1
	}
	fib := []int{1, 1}
	for i := 2; i <= n; i++ {
		next := fib[i-1] + fib[i-2]
		if next > maxInterval {
			return maxInterval
		}
		fib = append(fib, next)
	}
	return fib[n]
}
