package curl

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/magic-lib/go-plat-utils/cache"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"time"
)

// responseCacheStruct 返回的缓存结构
type responseCacheStruct struct {
	CreateTime time.Time `json:"createTime"`
	Response   string    `json:"response"`
}

func (g *genRequest) setDataToCache(ctx context.Context, p *Response) {
	if g.cacheTime == 0 || g.cli.cacheIns == nil {
		return
	}

	cacheId := p.Id
	if p.Id == "" {
		cacheId = getRequestId(p.Request)
	}

	goroutines.GoAsync(func(params ...interface{}) {
		cacheData := responseCacheStruct{
			CreateTime: time.Now(),
			Response:   p.Response,
		}
		cacheStr := conv.String(cacheData)
		if cacheStr == "" {
			return
		}

		_, _ = cache.NsSet[string](ctx, g.cli.cacheIns, cacheNamespace, cacheId, cacheStr, g.cacheTime)
	})
}

func (g *genRequest) getDataFromCache(ctx context.Context) string {
	if g.cacheTime == 0 || g.cli.cacheIns == nil {
		return ""
	}
	cacheId := getRequestId(g.getNewRequest())

	retData, err := cache.NsGet[string](ctx, g.cli.cacheIns, cacheNamespace, cacheId)
	if err != nil || retData == "" {
		return ""
	}

	cacheData := new(responseCacheStruct)
	err = jsoniter.Unmarshal([]byte(retData), cacheData)
	if err != nil {
		_, _ = cache.NsDel[string](ctx, g.cli.cacheIns, cacheNamespace, cacheId)
		return ""
	}
	//超时
	if time.Now().Sub(cacheData.CreateTime) > g.cacheTime {
		_, _ = cache.NsDel[string](ctx, g.cli.cacheIns, cacheNamespace, cacheId)
		return ""
	}
	return cacheData.Response
}
