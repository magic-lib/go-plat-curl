package curl_test

import "testing"

func TestCurlGrpcProxy(t *testing.T) {
	//cfgApi, err := startupcfg.NewByYamlFile("/Volumes/MacintoshData/workspace/goland-framework/code/magic-lib/go-plat-curl/curl/startup_cfg.yaml")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//grpcProxy, err := curl.NewGrpcProxy(&curl.GrpcConfig{
	//	ServiceConfig: cfgApi,
	//	ServiceName:   "member-server-grpc",
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//resp := new(memberPb.CompanyListResult)
	//retResp, err := grpcProxy.Submit(context.Background(), &curl.ProxyData{
	//	UrlCfgName: "CompanyList",
	//	CurlReq: &curl.Request{
	//		Data: &memberPb.CompanyListReq{
	//			PageNum:  1,
	//			PageSize: 10,
	//			KeyWord:  "",
	//			Sort:     1,
	//		},
	//	},
	//}, resp)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//t.Log(conv.String(retResp))
	//t.Log(conv.String(resp))
}
