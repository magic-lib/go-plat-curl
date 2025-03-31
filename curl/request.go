package curl

import (
	"net/http"
)

// Request 请求的实际参数
type Request struct {
	Url    string      `json:"url"`
	Data   interface{} `json:"data"`
	Method string      `json:"method"`
	Header http.Header `json:"header"`
}
