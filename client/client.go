package main

import (
	"context"
	xdsclient "go-control-plane-test/gRPC"
	"google.golang.org/grpc"
	"log"
)

// 纯grpcclient
func main() {
	conn, err := grpc.Dial("127.0.0.1:8888", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("连接错误: %v", err)
	}

	defer conn.Close()

	// 创建client
	client := xdsclient.NewXDSConfigClient(conn)

	log.Printf("开始传输")
	response, err := client.SetConfigToCache(context.Background(), &xdsclient.Require{
		ClusterName:     "example_proxy_cluster",
		UpstreamHost:    "www.envoyproxy.io",
		UpstreamPort:    80,
		RouteName:       "local_route",
		Domains:         []string{"*"},
		Prefix:          "/",
		ListenerName:    "listener_0",
		ListenerPort:    10000,
		GrpcClusterName: "xds_cluster",
	})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Printf("%v , code: %v", response.Response, response.Code)
}
