package curl

import (
	"errors"
	"github.com/magic-lib/go-plat-utils/conv"
	"io"
	"net/http"
	"time"
)

// Response 方法返回的变量，因为外部方法
type Response struct {
	Id         string        `json:"id"`
	Request    *Request      `json:"request"`
	Response   string        `json:"response"`
	Header     http.Header   `json:"header"`
	StatusCode int           `json:"status"`
	CostTime   time.Duration `json:"costTime"` //请求间隔时间
	Error      error         `json:"error"`
	fromCache  bool
	resp       *http.Response
	body       []byte
}

// setCostTime 设置间隔时间
func (r *Response) setCostTime(startTime time.Time) {
	r.CostTime = time.Now().Sub(startTime)
}

// setAndCloseHttpResp 设置http响应
func (r *Response) setAndCloseHttpResp(resp *http.Response) {
	if resp == nil {
		return
	}
	r.resp = resp
	r.Header = r.resp.Header
	r.StatusCode = r.resp.StatusCode
	body, err := r.setRespContent(resp)
	if err != nil {
		if r.Error == nil {
			r.Error = err
		}
	} else {
		r.Response = string(body)
	}
}

func (r *Response) setRespContent(resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}

	defer resp.Body.Close()
	if r.resp != nil {
		defer r.resp.Body.Close()
	}

	if len(r.body) > 0 {
		return r.body, nil
	}

	if resp == nil || resp.Body == nil {
		return nil, errors.New("response or body is nil")
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r.body = b
	return b, nil
}
func (r *Response) Unmarshal(v interface{}) error {
	if r.Error != nil {
		return r.Error
	}

	if r.Response == "" {
		return errors.New("response is empty")
	}

	return conv.Unmarshal(r.Response, &v)
}

func newResponse(req *Request) *Response {
	return &Response{
		Id:      getRequestId(req),
		Request: req,
	}
}
