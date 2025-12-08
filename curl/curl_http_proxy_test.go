package curl_test

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-curl/curl"
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"github.com/tidwall/gjson"
	"net/http"
	"time"
)
import "testing"

func TestCurlHttpProxy(t *testing.T) {
	cfgApi, err := startupcfg.NewByYamlFile("/Volumes/MacintoshData/workspace/goland-framework/code/magic-lib/go-plat-curl/curl/startup_cfg.yaml")
	if err != nil {
		t.Fatal(err)
	}

	biuMoney, err := NewBiuMoneyLogic(cfgApi, "")
	if err != nil {
		t.Fatal(err)
	}
	userInfo, err := biuMoney.UserInfoByMobile(nil, "0771990039")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(userInfo)
}

type BiuMoneyLogic struct {
	memberProxy *MemberMysqlProxy
}

type BiuMoneyUserInfo struct {
	FirstName        string `json:"FirstName"`
	LastName         string `json:"LastName"`
	IsOpenWallet     int    `json:"IsOpenWallet"`
	ZanacoKycResults any    `json:"ZanacoKycResults"`
	FnbKycResult     any    `json:"FnbKycResult"`
	ZamtelKycResult  any    `json:"ZamtelKycResult"`
}

type BiuMoneyCheckKYCResp struct {
	Data    *BiuMoneyUserInfo `json:"Data"`
	Total   int               `json:"Total"`
	Success bool              `json:"Success"`
	Code    int               `json:"Code"`
	Message string            `json:"Message"`
}

const (
	defaultCacheTime    = 24 * time.Hour
	biuMoneyServiceName = "biuMoney"
	mobileCheckKYC      = "MobileCheckKYC"
)

func NewBiuMoneyLogic(serviceConfig *startupcfg.ConfigAPI, mysqlDsn string) (*BiuMoneyLogic, error) {
	memberProxy, err := NewMemberMysqlProxy(serviceConfig, biuMoneyServiceName, mysqlDsn)
	if err != nil {
		return nil, err
	}
	return &BiuMoneyLogic{
		memberProxy: memberProxy,
	}, nil
}
func (b *BiuMoneyLogic) UserInfoByMobile(ctx context.Context, mobile string) (*BiuMoneyUserInfo, error) {
	respData := new(BiuMoneyCheckKYCResp)
	_, err := b.memberProxy.Submit(ctx, &curl.ProxyData{
		UrlCfgName: mobileCheckKYC,
		CurlReq: &curl.Request{
			Method: http.MethodGet,
			Data: map[string]any{
				"phonenumber": mobile,
			},
		},
		CacheConfig: &curl.ProxyCacheConfig{
			Namespace: mobileCheckKYC,
			CacheKey:  mobile,
			CacheTime: defaultCacheTime,
			CacheCheckFunc: func(resp *curl.Response) bool {
				return gjson.Get(resp.Response, "Code").Int() == 0 &&
					gjson.Get(resp.Response, "Data.FirstName").String() != ""
			},
		},
	}, respData)
	if err != nil {
		return nil, err
	}
	if respData.Code != 0 {
		return nil, fmt.Errorf("code: %d, message: %s", respData.Code, respData.Message)
	}
	return respData.Data, nil
}

const (
	mysqlCacheMemberTableName = "member_global_cache"
)

type MemberMysqlProxy struct {
	mysqlCache cache.CommCache[string]
	CurlProxy  curl.Proxy
}

func NewMemberMysqlProxy(serviceConfig *startupcfg.ConfigAPI, serviceName string, mysqlDsn string) (*MemberMysqlProxy, error) {
	curlProxy, err := curl.NewCurlProxy(&curl.ProxyConfig{
		ServiceConfig: serviceConfig,
		ServiceName:   serviceName,
	})
	if err != nil {
		return nil, err
	}
	var mysqlCache cache.CommCache[string]
	if mysqlDsn != "" {
		mysqlConfig := &cache.MySQLCacheConfig{
			DSN:            mysqlDsn,
			Namespace:      serviceName,
			CloseAutoClean: true,
			TableName:      mysqlCacheMemberTableName,
		}
		mysqlCache, err = cache.NewMySQLCache[string](mysqlConfig)
		if err != nil {
			return nil, err
		}
	}
	return &MemberMysqlProxy{
		CurlProxy:  curlProxy,
		mysqlCache: mysqlCache,
	}, nil
}

func (b *MemberMysqlProxy) Submit(ctx context.Context, proxyData *curl.ProxyData, dstPoint any) (*curl.Response, error) {
	if proxyData == nil {
		return nil, fmt.Errorf("proxyData is nil")
	}
	if proxyData.RetryConfig == nil {
		proxyData.RetryConfig = &curl.RetryPolicy{
			Attempts: 3,
		}
	}
	if b.mysqlCache != nil {
		if proxyData.CacheConfig == nil {
			proxyData.CacheConfig = &curl.ProxyCacheConfig{}
		}
		proxyData.CacheConfig.CachePool = b.mysqlCache
	}
	resp, err := b.CurlProxy.Submit(ctx, proxyData, dstPoint)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
