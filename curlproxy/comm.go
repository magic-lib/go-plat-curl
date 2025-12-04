package curlproxy

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-curl/curl"
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"github.com/magic-lib/go-plat-utils/conv"
	"net/http"
)

// CurlProxy 利用config，封装了curl的提交方法
type CurlProxy interface {
	ServerApi() startupcfg.ServiceAPI                                                                        //获取服务api配置
	Submit(ctx context.Context, urlName string, curlReq *curl.Request, dstPoint any) (*curl.Response, error) //提交数据
}

type Config struct {
	ServiceConfig *startupcfg.ConfigAPI
	ServiceName   string
}

type curlProxy struct {
	serverApi startupcfg.ServiceAPI
}

func NewCurlProxy(cfg *Config) (CurlProxy, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	if cfg.ServiceConfig == nil {
		return nil, fmt.Errorf("cfg.ServiceConfig is nil")
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("cfg.ServiceName is empty")
	}

	dataApi := cfg.ServiceConfig.ServiceAPI(cfg.ServiceName)
	if dataApi == nil {
		return nil, fmt.Errorf("%s not set", cfg.ServiceName)
	}
	return &curlProxy{
		serverApi: dataApi,
	}, nil
}

func (l *curlProxy) ServerApi() startupcfg.ServiceAPI {
	return l.serverApi
}

func (l *curlProxy) Submit(ctx context.Context, urlName string, curlReq *curl.Request, dstPoint any) (*curl.Response, error) {
	if curlReq == nil {
		curlReq = new(curl.Request)
	}
	if curlReq.Url == "" {
		curlReq.Url = fmt.Sprintf("%s%s", l.serverApi.DomainName(), l.serverApi.Url(urlName))
	}
	resp := curl.NewClient().NewRequest(curlReq).Submit(ctx)
	if resp.Error != nil {
		return resp, resp.Error
	}
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("%d:%s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	if dstPoint != nil {
		if err := conv.Unmarshal(resp.Response, dstPoint); err != nil {
			return resp, err
		}
		return resp, nil
	}
	return resp, nil
}
