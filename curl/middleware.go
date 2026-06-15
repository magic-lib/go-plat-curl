package curl

import (
	"context"
	"github.com/magic-lib/go-plat-utils/utils/httputil/param"
	"net/http"
	"time"

	"github.com/magic-lib/go-plat-utils/conv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// responseWriterWrapper 包装 http.ResponseWriter 以捕获状态码和响应体
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	body       []byte
	writerErr  error
}

// WriteHeader 重写以捕获状态码
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write 重写以捕获响应体
func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.body = append(w.body, data...)
	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		w.writerErr = err
	}
	return n, err
}

// HttpMiddlewareInterceptor 记录 http 请求和响应到日志表
// logCallback: 请求完成后回调，可在此将请求/响应数据写入日志或数据库
// 返回标准 HTTP 中间件，用法: router.Use(HttpMiddlewareInterceptor(myLogger))
func HttpMiddlewareInterceptor(logCallback func(ctx context.Context, respData *Response)) func(next http.HandlerFunc) http.HandlerFunc {
	if logCallback == nil {
		logCallback = func(ctx context.Context, respData *Response) {}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			var reqBodyByte = param.SafeReadBody(r)
			reqBodyStr := string(reqBodyByte)

			// 包装 ResponseWriter 以捕获响应数据
			wrappedWriter := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// 调用下一个 handler
			next(wrappedWriter, r)

			// 构建请求信息
			reqInfo := &Request{
				Method: r.Method,
				Url:    r.URL.String(),
				Data:   reqBodyStr,
				Header: r.Header,
			}

			// 构建响应数据
			respData := &Response{
				Request:    reqInfo,
				Response:   string(wrappedWriter.body),
				StatusCode: wrappedWriter.statusCode,
				CostTime:   time.Since(start),
			}
			if wrappedWriter.writerErr != nil {
				respData.Error = wrappedWriter.writerErr
			}

			logCallback(r.Context(), respData)
		}
	}
}

// GrpcMiddlewareInterceptor 记录 gRPC 请求和响应到日志表
func GrpcMiddlewareInterceptor(logCallback func(ctx context.Context, respData *Response)) grpc.UnaryServerInterceptor {
	if logCallback == nil {
		logCallback = func(ctx context.Context, respData *Response) {} // 默认什么也不做
	}
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		reqBody := conv.String(req)
		resp, err := handler(ctx, req)
		respBody := conv.String(resp)

		respData := &Response{
			Request: &Request{
				Method: "GRPC",
				Url:    info.FullMethod,
				Data:   reqBody,
			},
			Response:   respBody,
			StatusCode: int(status.Code(err)),
			CostTime:   time.Since(start),
		}
		if err != nil {
			respData.Error = err
		}
		logCallback(ctx, respData)
		return resp, err
	}
}
