package curl

import (
	"bytes"
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"time"
)

// 递归使用
func (g *genRequest) httpRequest(ctx context.Context, newUrl string, dataBuffer *bytes.Buffer, resp *Response) *Response {
	httpReq, err := g.getHttpRequest(ctx, newUrl, dataBuffer)
	if err != nil {
		resp.Error = err
		return resp
	}
	resp.Error = nil

	newRequest := g.getNewRequest()
	if g.cli.handler != nil {
		err = g.cli.handler.BeforeHandler(ctx, newRequest, httpReq)
		if err != nil {
			resp.Error = err
			return resp
		}
	}

	isRetry := false
	opts := make([]retry.Option, 0)
	if g.retryPolicy != nil && g.retryPolicy.Attempts > 0 {
		isRetry = true
		opts = g.retryPolicy.getRetryOptions()
	}

	startTime := time.Now()

	if !isRetry {
		retResp, err := g.requestDo(httpReq, resp)
		retResp, err = g.requestDoBack(ctx, startTime, retResp, err)
		if err != nil {
			retResp.Error = err
		}
		return retResp
	}

	var retRespTemp *Response

	//需要重试
	retResp, err := retry.DoWithData[*Response](func() (*Response, error) {
		respTemp, err := g.requestDo(httpReq, resp)
		if respTemp != nil {
			retRespTemp = respTemp
			logStr := fmt.Sprintf("[comm-request http-request retry.do]id:%s, error:%v", respTemp.Id, err)
			printLog(ctx, g.cli.logger, 0, g.defaultPrintLogInt, logStr)
		}

		if err != nil {
			return respTemp, err
		}
		//自定义需要重试的函数，可能业务需要重试
		if g.retryPolicy != nil {
			err = g.retryPolicy.hasRetryError(respTemp)
			if err != nil {
				return respTemp, err
			}
		}

		return respTemp, nil
	}, opts...)

	if retResp == nil {
		retResp = retRespTemp
	}
	if retResp == nil {
		retResp = resp
	}

	if err != nil {
		retResp.Error = err
	}

	retResp, err = g.requestDoBack(ctx, startTime, retResp, err)
	if err != nil {
		retResp.Error = err
	}

	return retResp
}
