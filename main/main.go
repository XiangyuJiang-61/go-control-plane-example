// Copyright 2020 Envoyproxy Authors
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	    http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.
package main

import (
	"context"
	"flag"
	example "go-control-plane-test"
	"go-control-plane-test/xdsserver"
	"os"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
)

var (
	l      example.Logger
	port   uint
	nodeID string
)

// 初始化的时候读取启动指令里面flag的值，没有就用默认值
func init() {
	l = example.Logger{}

	flag.BoolVar(&l.Debug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")
}

func main() {
	// 解析flag里面的值，填充到结构体里面
	flag.Parse()

	// Create a cache
	// 创建缓存
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Create the snapshot that we'll serve to Envoy
	// 创建要传给envoy的快照
	// 这里是直接读了resource包的数据，生成了一个带有xDS信息的快照，能根据xDS资源类型（例如CDS、EDS）和对应资源的name去索引对应资源，支持TTL
	snapshot := example.DefaultSnapshot()
	// 快照依赖检查，包括各个xDS资源间的引用是否被声明，这里只检查了EDS和RDS是否被正确引用，确保快照一致性
	if err := snapshot.Consistent(); err != nil {
		l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		os.Exit(1)
	}
	l.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	// 把快照放到缓存里面
	// 会监视当前传入的nodeid下的这个快照，如果更新了就触发，并且更新现在监控的对象
	// think:应该就是这玩意处理传入的值，那么怎么传入呢？
	if err := cache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
		l.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}

	// Run the xDS server
	// 起xDS server
	// 这块是server
	ctx := context.Background()
	// 传入回调函数，确定日志级别是否是debug，test包是调试用的工具，包括debug级别日志，这个回调函数主要就是用来打印日志的
	cb := &test.Callbacks{Debug: l.Debug}
	// 这里是配置server，传入了缓存，初始上下文和调试相关配置
	srv := server.NewServer(ctx, cache, cb)
	// 开始准备后续的部分，通过一个gRPC去接收信息
	go xdsserver.RunServer("8888", nodeID, cache)

	// 调了server的方法起服务了
	example.RunServer(ctx, srv, port)

}
