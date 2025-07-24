package forge_connect

import "time"

// Response defines the unified API response structure
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Task defines the basic task structure
type Task struct {
	TaskID   string    `json:"task_id"`   // Unique task identifier (UUID)
	TaskType string    `json:"task_type"` // Task type
	DoStatus string    `json:"do_status"`
	CreateAt time.Time `json:"create_at"`
	Payload  string    `json:"payload"` // Task-specific data (e.g., JSON)
	Result   string    `json:"result"`
}

type RegistrationRequest struct {
	AppID  string `json:"app_id"`
	Secret string `json:"secret"`
}

// API routes shared by client and server
var apiRoutes = map[string]string{
	"register":   "/orange-forge/api/register",
	"ping":       "/orange-forge/api/ping",
	"getTask":    "/orange-forge/api/getTask",
	"reportTask": "/orange-forge/api/reportTask",
}

const (
	SUCCESS        = 0
	STATUS_DOING   = "doing"
	STATUS_TIMEOUT = "timeout"
	STATUS_SUCCESS = "success"

	LOG_TYPE    = "logging"
	ERR_TYPE    = "error"
	EXPIRE_TYPE = "timeout"
	SUCC_TYPE   = "success"

	RDX_EXPIRE = 604800

	DEFAULT_SECRET = "orange-forge"
)
