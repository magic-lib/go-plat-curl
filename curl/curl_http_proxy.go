package curl

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"github.com/magic-lib/go-plat-utils/conv"
	"net/http"
	"time"
)

type ProxyData struct {
	UrlCfgName      string   //url配置名称
	CurlReq         *Request //curl请求参数,GET\POST
	Timeout         time.Duration
	CacheConfig     *ProxyCacheConfig                      //缓存配置
	RetryConfig     *RetryPolicy                           //重试配置
	BuildReqHandler func(rb RequestBuilder) RequestBuilder //构建请求的函数，特殊处理，一般用不上
}

type httpProxy struct {
	serverApi startupcfg.ServiceAPI
}

func NewHttpProxy(cfg *ProxyConfig) (Proxy, error) {
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
	return &httpProxy{
		serverApi: dataApi,
	}, nil
}

func (l *httpProxy) ServerApi() startupcfg.ServiceAPI {
	return l.serverApi
}

func (l *httpProxy) Submit(ctx context.Context, proxyData *ProxyData, dstPoint any) (*Response, error) {
	if proxyData == nil {
		return nil, fmt.Errorf("proxyData is nil")
	}

	proxyData = buildCurlReq(l.serverApi, proxyData)
	proxyData = buildCurlProxyRetry(proxyData)
	proxyData = buildCurlProxyCache(proxyData)

	var rb RequestBuilder = NewClient().NewRequest(proxyData.CurlReq)
	if proxyData.RetryConfig != nil {
		rb = rb.SetRetryPolicy(proxyData.RetryConfig)
	}
	if proxyData.Timeout > 0 {
		rb = rb.SetTimeout(proxyData.Timeout)
	}
	if proxyData.BuildReqHandler != nil {
		rb = proxyData.BuildReqHandler(rb)
	}

	allRetStr := ""
	useCache := checkOpenCache(proxyData.CacheConfig)
	if useCache {
		retStr, err := cache.NsGet(ctx, proxyData.CacheConfig.CachePool, proxyData.CacheConfig.Namespace, proxyData.CacheConfig.CacheKey)
		allRetStr = retStr              // 保存可能过期的数据
		if err == nil && retStr != "" { //表示为过期数据
			retResp := &Response{
				Request:    proxyData.CurlReq,
				StatusCode: http.StatusOK,
				Response:   retStr,
			}
			if dstPoint == nil {
				return retResp, nil
			} else {
				if err = conv.Unmarshal(retStr, dstPoint); err == nil {
					return retResp, nil
				}
			}
		}
	}

	resp := rb.Submit(ctx)

	useExpireDataFunc := func(resp *Response, outErr error) (*Response, error) {
		if proxyData.CacheConfig == nil {
			return resp, outErr
		}
		if !proxyData.CacheConfig.CacheDontUseExpire && allRetStr != "" {
			resp.Response = allRetStr
			resp.Error = nil
			resp.StatusCode = http.StatusOK
			if dstPoint != nil {
				if err := conv.Unmarshal(resp.Response, dstPoint); err != nil {
					return resp, err
				}
			}
			return resp, nil
		}
		return resp, outErr
	}

	if resp.Error != nil {
		return useExpireDataFunc(resp, resp.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return useExpireDataFunc(resp, fmt.Errorf("%d:%s", resp.StatusCode, http.StatusText(resp.StatusCode)))
	}

	var cacheFunction = func() {
		if useCache {
			// 如果满足缓存条件，则缓存数据
			if proxyData.CacheConfig.CacheCheckFunc(resp) {
				_, _ = cache.NsSet(ctx,
					proxyData.CacheConfig.CachePool,
					proxyData.CacheConfig.Namespace,
					proxyData.CacheConfig.CacheKey,
					resp.Response, proxyData.CacheConfig.CacheTime)
			}
		}
	}

	if dstPoint != nil {
		if err := conv.Unmarshal(resp.Response, dstPoint); err != nil {
			return resp, err
		}
	}
	cacheFunction()

	return resp, nil
}

func checkOpenCache(cacheData *ProxyCacheConfig) bool {
	if cacheData == nil {
		return false
	}
	if cacheData.Namespace != "" &&
		cacheData.CacheKey != "" &&
		cacheData.CacheTime > 0 &&
		cacheData.CacheCheckFunc != nil &&
		cacheData.CachePool != nil {
		return true
	}
	return false
}
