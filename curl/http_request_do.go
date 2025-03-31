package curl

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"net/http"
	"time"
)

// requestDo 发起请求
func (g *genRequest) requestDo(httpReq *http.Request, retResp *Response) (*Response, error) {
	if retResp == nil {
		retResp = newResponse(g.getNewRequest())
	}

	resp, err := g.cli.httpCli.Do(httpReq)
	if err != nil {
		retResp.Error = err
		return retResp, err
	}

	retResp.setAndCloseHttpResp(resp)

	return retResp, nil
}

// requestDoBack 执行完以后的方法
func (g *genRequest) requestDoBack(ctx context.Context, startTime time.Time, retResp *Response, err error) (*Response, error) {
	retResp.setCostTime(startTime)

	logStr := fmt.Sprintf("[comm-request http-request return]id:%s, error:%v", retResp.Id, err)
	printLog(ctx, g.cli.logger, 0, g.defaultPrintLogInt, logStr)

	if g.cli.handler != nil {
		err = g.cli.handler.AfterHandler(ctx, retResp)
		if err != nil {
			retResp.Error = err
			return retResp, err
		}
	}

	//如果设置了返回的类型，则可以进行判断
	if g.respDateType == respDataTypeJson {
		if retResp.Error == nil {
			var obj interface{}
			err = jsoniter.Unmarshal([]byte(retResp.Response), &obj)
			if err != nil {
				//返回的不是json格式
				retResp.Error = fmt.Errorf("url: %s, response not json: %s，respDateType=string", retResp.Request.Url, retResp.Response)
				return retResp, retResp.Error
			}
		}
	}

	return retResp, nil
}
