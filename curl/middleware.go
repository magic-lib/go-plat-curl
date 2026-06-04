package curl

import (
	"context"
	"github.com/magic-lib/go-plat-utils/conv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"time"
)

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
