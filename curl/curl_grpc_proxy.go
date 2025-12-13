package curl

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"github.com/magic-lib/go-plat-utils/conv"
	"google.golang.org/grpc"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type grpcProxy struct {
	serverApi   startupcfg.ServiceAPI
	connPool    *cache.CommPool[*grpc.ClientConn]
	opts        []grpc.DialOption
	connPoolCfg *cache.ResPoolConfig[*grpc.ClientConn]
}

func NewGrpcProxy(cfg *GrpcConfig, opt ...grpc.DialOption) (Proxy, error) {
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

	connPool, err := getGrpcConnPool(dataApi.DomainName(), cfg.ConnPoolCfg, opt...)
	if err != nil {
		return nil, err
	}
	return &grpcProxy{
		serverApi:   dataApi,
		connPool:    connPool,
		connPoolCfg: cfg.ConnPoolCfg,
		opts:        opt,
	}, nil
}

func (l *grpcProxy) ServerApi() startupcfg.ServiceAPI {
	return l.serverApi
}

func (l *grpcProxy) getConn() (*grpc.ClientConn, error) {
	oneConn, err := l.connPool.Get()
	if err != nil {
		return nil, err
	}
	return oneConn.Resource, nil
}

func (l *grpcProxy) Submit(ctx context.Context, proxyData *ProxyData, dstPoint any) (*Response, error) {
	if proxyData == nil {
		return nil, fmt.Errorf("grpcData is nil")
	}
	if proxyData.Timeout <= 0 {
		proxyData.Timeout = 30 * time.Second
	}

	proxyData = buildCurlReq(l.serverApi, proxyData)
	proxyData = buildCurlProxyCache(proxyData)
	proxyData = buildCurlProxyRetry(proxyData)

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

	var cacheFunction = func(resp *Response) {
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

	retResp := newResponse(proxyData.CurlReq)

	oneConn, methodName, req, reqErr := l.getGrpcRequest(proxyData.CurlReq)
	if reqErr != nil {
		return useExpireDataFunc(retResp, reqErr)
	}
	if oneConn == nil {
		return useExpireDataFunc(retResp, fmt.Errorf("getConn is nil"))
	}

	var connCtx = ctx
	if proxyData.Timeout > 0 {
		var cancel context.CancelFunc
		connCtx, cancel = context.WithTimeout(ctx, proxyData.Timeout)
		defer cancel()
	}

	isRetry := false
	opts := make([]retry.Option, 0)
	if proxyData.RetryConfig != nil && proxyData.RetryConfig.Attempts > 0 {
		isRetry = true
		opts = proxyData.RetryConfig.getRetryOptions()
	}

	startTime := time.Now()

	if !isRetry {
		err := l.submit(connCtx, oneConn, methodName, retResp, req, dstPoint)
		retResp.CostTime = time.Now().Sub(startTime)
		if err != nil {
			retResp.Error = err
			return useExpireDataFunc(retResp, err)
		}
		cacheFunction(retResp)
		return useExpireDataFunc(retResp, nil)
	}

	var err error
	retResp, err = retry.DoWithData[*Response](func() (*Response, error) {
		err = l.submit(connCtx, oneConn, methodName, retResp, req, dstPoint)
		if err != nil {
			retResp.Error = err
			return retResp, err
		}

		if proxyData.RetryConfig != nil {
			err = proxyData.RetryConfig.hasRetryError(retResp)
			if err != nil {
				return retResp, err
			}
		}
		return retResp, nil
	}, opts...)

	retResp.CostTime = time.Now().Sub(startTime)
	if err != nil {
		retResp.Error = err
		return useExpireDataFunc(retResp, err)
	}
	cacheFunction(retResp)
	return useExpireDataFunc(retResp, nil)
}

func (l *grpcProxy) getGrpcRequest(curlReq *Request) (conn *grpc.ClientConn, method string, req any, err error) {
	if curlReq != nil {
		if curlReq.Url != "" {
			var domain = ""
			domain, method, err = parseAddress(curlReq.Url)
			if err != nil {
				return
			}
			connPool, poolErr := getGrpcConnPool(domain, l.connPoolCfg, l.opts...)
			if poolErr != nil {
				err = poolErr
				return
			}
			connRes, poolErr := connPool.Get()
			if poolErr != nil {
				err = poolErr
				return
			}
			conn = connRes.Resource
		}
		req = curlReq.Data
	}

	if conn == nil {
		conn, err = l.getConn()
		if err != nil {
			return
		}
	}
	if method == "" {
		return
	}
	return
}

func (l *grpcProxy) submit(connCtx context.Context, conn *grpc.ClientConn, methodName string, retResp *Response, req, resp any) error {
	if conn == nil {
		return fmt.Errorf("conn is nil")
	}
	//cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, callOpts...)
	err := conn.Invoke(connCtx, methodName, req, resp)
	if err != nil {
		return err
	}
	retResp.StatusCode = http.StatusOK
	retResp.Response = conv.String(resp)
	return nil
}

func parseAddress(rawAddr string) (domain string, path string, err error) {
	if !strings.HasPrefix(rawAddr, "http://") && !strings.HasPrefix(rawAddr, "https://") {
		rawAddr = "http://" + rawAddr
	}

	parsedURL, err := url.Parse(rawAddr)
	if err != nil {
		return "", "", fmt.Errorf("解析地址失败: %v", err)
	}
	domain = parsedURL.Host
	path = parsedURL.Path

	return domain, path, nil
}
