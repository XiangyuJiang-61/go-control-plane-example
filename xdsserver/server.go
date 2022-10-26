package xdsserver

import (
	"context"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xdsclient "go-control-plane-test/gRPC"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)
import example "go-control-plane-test"

type server struct {
	cache  cache.SnapshotCache
	nodeID string
}

// SetConfigToCache 执行业务逻辑，取出对应部分并存入缓存
func (s *server) SetConfigToCache(ctx context.Context, require *xdsclient.Require) (*xdsclient.Response, error) {
	xDSRequest := example.XDSRequest{
		ClusterName:     require.ClusterName,
		UpstreamHost:    require.UpstreamHost,
		UpstreamPort:    require.UpstreamPort,
		RouteName:       require.RouteName,
		Domains:         require.Domains,
		Prefix:          require.Prefix,
		ListenerName:    require.ListenerName,
		ListenerPort:    require.ListenerPort,
		GrpcClusterName: require.GrpcClusterName,
	}

	err := s.cache.SetSnapshot(context.Background(), s.nodeID, example.GenerateSnapshot(xDSRequest))
	if err != nil {
		log.Printf("写入缓存错误: %v", err)
		return &xdsclient.Response{
			Response: "failed",
			Code:     500,
		}, err
	}

	return &xdsclient.Response{
		Response: "success",
		Code:     200,
	}, nil
}

func RunServer(port string, nodeID string, cache cache.SnapshotCache) {
	log.Printf("开始client监听，端口: %v", port)
	lin, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("监听失败: %v", err)
		os.Exit(1)
	}

	// 创建并注册
	g := grpc.NewServer()
	xdsclient.RegisterXDSConfigServer(g, &server{nodeID: nodeID, cache: cache})
	// 起服务
	err = g.Serve(lin)
	if err != nil {
		log.Fatalf("server failed: %v", err)
		os.Exit(1)
	}

}
