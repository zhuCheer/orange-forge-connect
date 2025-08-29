package forge_connect

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	rdx "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

// Define a channel to hold the payloads
var payloadChannel = make(chan string, 500) // Buffered channel with a capacity of 100

type ClientInfo struct {
	AppID              string `json:"app_id"`
	Secret             string `json:"secret"`
	RegisterTime       int64  `json:"register_time"`
	LastPingTime       int64  `json:"last_ping_time"`
	DoStatus           string `json:"do_status"`
	ProcessedTaskCount int    `json:"processed_task_count"`
}

type Server struct {
	redisConn        rdx.Conn
	IsDebug          bool
	ServerName       string
	SessionId        string
	RunAt            time.Time
	httpMux          *http.ServeMux
	statusFunc       func(i Task)
	mutex            sync.Mutex
	singleTimeout    time.Duration
	longLoopDuration time.Duration
	taskChan         map[string]chan Task
	taskWaitTick     time.Duration
}

func NewServer(serverName string) *Server {
	return &Server{
		ServerName:       serverName,
		RunAt:            time.Now(),
		SessionId:        uuid.New().String(),
		statusFunc:       listenTaskStatus,
		singleTimeout:    30 * time.Second,
		longLoopDuration: 10 * time.Second,
		taskChan:         make(map[string]chan Task),
		taskWaitTick:     1 * time.Second,
	}
}

// SetDebug set Debug version show more logs
func (s *Server) SetDebug() *Server {
	s.IsDebug = true
	return s
}

// SetTaskWaitTick set task long loop tick duration
func (s *Server) SetTaskWaitTick(duration time.Duration) *Server {
	s.taskWaitTick = duration
	return s
}

// WithRdx add the fresh redis connect
func (s *Server) WithRdx(conn rdx.Conn) *Server {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.redisConn = conn
	return s
}

// WithSingleTimeout set single task timeout duration
func (s *Server) WithSingleTimeout(timeout time.Duration) *Server {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.singleTimeout = timeout
	return s
}

// WithRdx add the fresh redis connect
func (s *Server) WithStatusFunc(statusFunc func(i Task)) *Server {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	//go s.processPayloads()
	s.statusFunc = statusFunc
	return s
}

// ContinuousTask After Execution Completes, Asynchronously Receive Messages (e.g., Query Logs, Execute Commands)
func (s *Server)ContinuousTask(appID , taskType, payload string)(err error){

}

// RunSingleTask quickly send a task to the specified appid client and wait for the return
func (s *Server) RunSingleTask(appID, taskType, payload string) (taskID, respBody string, err error) {
	err = s.AppLiveCheck(appID)
	if err != nil {
		return
	}
	taskID, err = s.addTask(appID, taskType, payload)
	sttm := time.Now()
	if s.IsDebug {
		log.Println("[DEBUG] add task:", taskID)
	}

	// 创建独立的结果通道
	resultChan := make(chan Task, 1)
	defer func() {
		delete(s.taskChan, taskID) // 删除任务通道映射
	}()

	s.mutex.Lock()
	s.taskChan[taskID] = resultChan // 将通道映射到任务ID
	s.mutex.Unlock()

	// 等待任务结果或超时
	select {
	case task := <-resultChan:
		return taskID, task.Result, nil
	case <-time.After(s.singleTimeout):
		during := time.Since(sttm)
		if s.IsDebug {
			consoleLog("DEBUG", "task listen timeout during: %v， taskID: %v", during, taskID)
		}
		return taskID, "", fmt.Errorf("timeout waiting for task %s", taskID)
	}
}

// AppLiveCheck check app connect status
func (s *Server) AppLiveCheck(appID string) (err error) {
	err = s.verifyOpts()
	if err != nil {
		return
	}
	cacheKey := GetClientInfoKey(appID)
	clientJson, _ := rdx.String(s.redisConn.Do("GET", cacheKey))
	clientInfo := ClientInfo{}
	json.Unmarshal([]byte(clientJson), &clientInfo)
	now := time.Now().Unix()
	if clientInfo.AppID == "" {
		return errors.New("not found app info")
	}
	sincTm := now - clientInfo.LastPingTime
	if sincTm > 90 {
		clientInfo.DoStatus = STATUS_TIMEOUT
		infoJSON, _ := json.Marshal(clientInfo)
		_, err = s.redisConn.Do("SETEX", cacheKey, RDX_EXPIRE, infoJSON)

		return errors.New("the client is disconnected for more than 300 seconds")
	}
	return nil
}

// registerHandler handles client registration by reading the full request body,
// verifying the signature (which includes the body content and a date header),
// and storing client info and metadata in Redis.
func (s *Server) apiRegisterHandler(w http.ResponseWriter, r *http.Request) {
	err := s.verifyOpts()
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}
	providedSign, appID, dateTime, payload, err := getRequestArgs(r)
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}

	// register function is the secret use default:orange-forge
	expectedSign := s.computeSignature(appID, DEFAULT_SECRET, payload, dateTime)
	if expectedSign != providedSign {
		s.errorReport(w, 1, "signature verification failed")
		return
	}

	var req RegistrationRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		s.errorReport(w, 1, "invalid JSON")
		return
	}
	if req.AppID == "" || req.Secret == "" {
		s.errorReport(w, 1, "app_id and secret are required")
		return
	}
	cacheKey := GetClientInfoKey(req.AppID)
	clientJson, _ := rdx.String(s.redisConn.Do("GET", cacheKey))

	now := time.Now().Unix()
	clientInfo := ClientInfo{
		AppID:              req.AppID,
		Secret:             req.Secret,
		RegisterTime:       now,
		LastPingTime:       now,
		DoStatus:           "registered",
		ProcessedTaskCount: 0,
	}

	if clientJson != "" {
		json.Unmarshal([]byte(clientJson), &clientInfo)
		clientInfo.AppID = req.AppID
		clientInfo.Secret = req.Secret
		clientInfo.LastPingTime = now
	}

	infoJSON, err := json.Marshal(clientInfo)
	if err != nil {
		s.errorReport(w, 1, "failed to marshal client info")
		return
	}
	_, err = s.redisConn.Do("SETEX", GetClientInfoKey(req.AppID), RDX_EXPIRE, infoJSON)
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}

	//  hide secret for response
	clientInfo.Secret = "***"
	writeJSON(w, Response{Code: 0, Message: "registration successful", Data: clientInfo})
}

// pingHandler handles client pings by verifying the signature and updating the client's last ping time.
func (s *Server) apiPingHandler(w http.ResponseWriter, r *http.Request) {
	providedSign, appID, dateTime, reqBody, err := getRequestArgs(r)
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}
	if !s.verifySignature(appID, reqBody, dateTime, providedSign) {
		s.errorReport(w, 1, "signature verification failed")
		return
	}
	if s.IsDebug {
		log.Println("[debug] pingHandler", appID, reqBody, dateTime)
	}

	cacheKey := GetClientInfoKey(appID)
	clientJson, _ := rdx.String(s.redisConn.Do("GET", cacheKey))
	clientInfo := ClientInfo{}
	now := time.Now().Unix()

	if clientJson != "" {
		_ = json.Unmarshal([]byte(clientJson), &clientInfo)
		clientInfo.LastPingTime = now
		clientInfo.DoStatus = "registered"
	}
	infoJSON, err := json.Marshal(clientInfo)
	if err != nil {
		s.errorReport(w, 1, "failed to marshal client info")
		return
	}
	_, err = s.redisConn.Do("SETEX", GetClientInfoKey(appID), RDX_EXPIRE, infoJSON)

	writeJSON(w, Response{Code: 0, Message: "pong", Data: "pong"})
	return
}

// apiPushTaskStatus client return task information
func (s *Server) apiPushTaskStatus(w http.ResponseWriter, r *http.Request) {
	providedSign, appID, dateTime, reqBody, err := getRequestArgs(r)
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}
	if !s.verifySignature(appID, reqBody, dateTime, providedSign) {
		s.errorReport(w, 1, "signature verification failed")
		return
	}
	taskReciveData := Task{}
	_ = json.Unmarshal([]byte(reqBody), &taskReciveData)
	if taskReciveData.TaskID == "" {
		writeJSON(w, Response{Code: 1, Message: "task payload not found"})
		return
	}

	taskKey := "client:" + appID + ":task:" + taskReciveData.TaskID
	taskJSON, err := rdx.Bytes(s.redisConn.Do("GET", taskKey))
	if err != nil {
		s.errorReport(w, 1, "task info not found,"+err.Error())
		return
	}
	saveTaskInfo := Task{}
	_ = json.Unmarshal(taskJSON, &saveTaskInfo)
	saveTaskInfo.DoStatus = taskReciveData.DoStatus

	saveTaskInfoJson, _ := json.Marshal(saveTaskInfo)
	_, err = s.redisConn.Do("SETEX", taskKey, RDX_EXPIRE, saveTaskInfoJson)
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}

	if taskReciveData.DoStatus != STATUS_DOING {
		// remove task process key
		procQueueKey := "client:" + appID + ":processing_queue"
		s.redisConn.Do("LREM", procQueueKey, 1, taskReciveData.TaskID)
		if s.IsDebug {
			log.Println("[DEBUG] LREM:", procQueueKey, taskReciveData.TaskID)
		}
	}

	if _, ok := s.taskChan[taskReciveData.TaskID]; ok {
		s.taskChan[taskReciveData.TaskID] <- taskReciveData
	}

	writeJSON(w, Response{Code: 0, Message: "task status updated successfully"})
	return
}

// apiGetTaskHandler implements long-polling to fetch tasks.
// It first attempts an immediate RPOPLPUSH; if no task is available,
// it subscribes to the client's task channel and waits up to x seconds.
// When a notification is received, it attempts to fetch and lock a task.
func (s *Server) apiGetTaskHandler(w http.ResponseWriter, r *http.Request) {
	providedSign, appID, dateTime, payload, err := getRequestArgs(r)
	if err != nil {
		s.errorReport(w, 1, err.Error())
		return
	}
	if !s.verifySignature(appID, payload, dateTime, providedSign) {
		s.errorReport(w, 1, "signature verification failed")
		return
	}

	queueKey := "client:" + appID + ":task_queue"
	procQueueKey := "client:" + appID + ":processing_queue"

	ctx, cancel := context.WithTimeout(context.Background(), s.longLoopDuration)
	defer cancel()

	// 尝试立即获取任务
	taskID, err := rdx.String(s.redisConn.Do("RPOPLPUSH", queueKey, procQueueKey))
	if err == nil && taskID != "" {
		s.processTask(w, appID, taskID, procQueueKey)
		return
	}

	// 创建独立的结果通道
	taskResultChan := make(chan string, 1)
	// 启动轮询协程
	go func() {
		ticker := time.NewTicker(s.taskWaitTick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				taskID, err := rdx.String(s.redisConn.Do("RPOPLPUSH", queueKey, procQueueKey))
				if err == nil && taskID != "" {
					// 非阻塞发送
					taskResultChan <- taskID
					return
				}
			}
		}
	}()

	// 等待任务结果或超时
	select {
	case tid := <-taskResultChan:
		s.processTask(w, appID, tid, procQueueKey)
		return
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.errorReport(w, 2, "timeout reached without receiving a task")
			return
		}
		s.errorReport(w, 1, "unexpected error")
		return
	}
}

// addTask creates a new task for a specific client, stores it in Redis, and pushes its taskID into the client's task queue.
func (s *Server) addTask(appID, taskType, payload string) (string, error) {
	taskID := uuid.New().String()
	task := Task{
		TaskID:   taskID,
		TaskType: taskType,
		CreateAt: time.Now(),
		Payload:  payload,
	}
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return "", err
	}
	taskKey := "client:" + appID + ":task:" + taskID
	if _, err = s.redisConn.Do("SETEX", taskKey, RDX_EXPIRE, taskJSON); err != nil {
		return "", err
	}
	queueKey := "client:" + appID + ":task_queue"
	if _, err = s.redisConn.Do("LPUSH", queueKey, taskID); err != nil {
		return "", err
	}
	return taskID, err
}

// processTask attempts to retrieve task details, acquire a lock, and return the task.
func (s *Server) processTask(w http.ResponseWriter, appID, taskID, procQueueKey string) {
	taskKey := "client:" + appID + ":task:" + taskID
	taskJSON, err := rdx.Bytes(s.redisConn.Do("GET", taskKey))
	if err != nil {
		s.errorReport(w, 1, err.Error())

		return
	}
	var task Task
	if err = json.Unmarshal(taskJSON, &task); err != nil {
		s.errorReport(w, 1, "invalid task data")
		return
	}

	// Use SETNX to acquire a lock for the task.
	lockKey := "lock:client:" + appID + ":task:" + task.TaskID
	locked, err := rdx.Int(s.redisConn.Do("SETNX", lockKey, "1"))
	if err != nil || locked != 1 {
		// If lock not acquired, remove the task from the processing queue.
		s.redisConn.Do("LREM", procQueueKey, 1, taskID)
		if s.IsDebug {
			log.Println("[DEBUG] LREM:", procQueueKey)
		}
		//w.WriteHeader(http.StatusNoContent)
		s.errorReport(w, 2, "task is already being processed or lock acquisition failed")
		return
	}
	// Set an expiration for the lock (e.g., 120 seconds).
	s.redisConn.Do("EXPIRE", lockKey, 120)
	writeJSON(w, Response{Code: 0, Message: "task fetched", Data: task})
}

// computeSignature calculates HMAC-SHA256 signature using appID, payload, and dateTime.
func (s *Server) computeSignature(appID, secret, payload, dateTime string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(appID + payload + dateTime))
	sign := hex.EncodeToString(h.Sum(nil))
	if s.IsDebug {
		log.Println("signed before:", appID+payload+dateTime)
		log.Println("sign string:", sign)
	}
	return sign
}

// verifySignature retrieves the client's secret from Redis, validates the timestamp,
// and compares the expected signature with the provided one.
// The dateTime must be in the format "2006-01-02 15:04:05" and within a +/-5 minutes window.
func (s *Server) verifySignature(appID, payload, dateTime, providedSign string) bool {
	t := DateToTm(dateTime)

	now := time.Now()
	if now.Sub(t) > 5*time.Minute || t.Sub(now) > 5*time.Minute {
		if s.IsDebug {
			log.Println("[debug] requset datetime invalid", appID, dateTime)
		}
		return false
	}

	clientInfo, err := s.refreshClientInfo(appID)
	if err != nil {
		consoleRouter("[ERROR]", fmt.Sprintf("refreshClientInfo error: %v", err))
		return false
	}

	expectedSign := s.computeSignature(clientInfo.AppID, clientInfo.Secret, payload, dateTime)
	if s.IsDebug {
		log.Printf("[debug] expectedSign: %v, input:%v  ismatch:%v", expectedSign, providedSign, expectedSign == providedSign)
	}
	return expectedSign == providedSign
}

// refreshClientInfo get client info and refresh status
func (s *Server) refreshClientInfo(appID string) (info ClientInfo, err error) {
	clientKey := GetClientInfoKey(appID)
	clientInfoJSON, err := rdx.Bytes(s.redisConn.Do("GET", clientKey))
	if err != nil {
		return
	}
	if err = json.Unmarshal(clientInfoJSON, &info); err != nil {
		return
	}

	info.LastPingTime = time.Now().Unix()

	infoJSON, err := json.Marshal(info)
	if err != nil {
		return
	}
	_, err = s.redisConn.Do("SETEX", GetClientInfoKey(appID), 86400, infoJSON)
	if err != nil {
		return
	}
	return
}

func (s *Server) processPayloads() {
	for payload := range payloadChannel {
		// Process the payload (e.g., log it, update task status, etc.)
		fmt.Println("Processing payload:", payload)
		taskInfo := Task{}
		json.Unmarshal([]byte(payload), &taskInfo)
		s.statusFunc(taskInfo)
	}
}

func (s *Server) verifyOpts() (err error) {
	if s.redisConn == nil {
		err = errors.New("redis connection not found")
	}
	if s.statusFunc == nil {
		err = errors.New("must register StatusFunc use WithStatusFunc")
	}

	return
}

// Server_errorReport reports an error to the client with a specific HTTP status code and message.
func (s *Server) errorReport(w http.ResponseWriter, code int, message string) {
	if !s.IsDebug {
		message = "internal server error"
	}

	writeJSON(w, Response{Code: code, Message: message})
}

func getRequestArgs(r *http.Request) (providedSign, appID, dateTime, payload string, err error) {
	providedSign = r.Header.Get("X-FORGE-SIGN")
	appID = r.Header.Get("X-FORGE-APPID")
	dateTime = r.Header.Get("X-FORGE-TIME")
	if appID == "" || providedSign == "" || dateTime == "" {
		err = errors.New("appid, sign, and time are required")
		return
	}

	// Read the full request body to use as payload for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = errors.New("failed to read request body")
		return
	}
	payload = string(body)

	return
}

// writeJSON
func writeJSON(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func listenTaskStatus(i Task) {
	fmt.Println("===================", i)
}

// Handler returns a http.Handler with all API routes registered.
func (s *Server) Handler() http.Handler {
	if s.httpMux != nil {
		return s.httpMux
	}

	mux := http.NewServeMux()
	mux.HandleFunc(apiRoutes["register"], s.apiRegisterHandler)
	consoleRouter("POST", apiRoutes["register"])

	mux.HandleFunc(apiRoutes["ping"], s.apiPingHandler)
	consoleRouter("POST", apiRoutes["ping"])

	mux.HandleFunc(apiRoutes["getTask"], s.apiGetTaskHandler)
	consoleRouter("POST", apiRoutes["getTask"])

	mux.HandleFunc(apiRoutes["reportTask"], s.apiPushTaskStatus)
	consoleRouter("POST", apiRoutes["reportTask"])

	//mux.HandleFunc(apiRoutes["reportTask"], s.reportTaskHandler)
	s.httpMux = mux
	return mux
}
