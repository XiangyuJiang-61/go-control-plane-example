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
package example

import (
	"time"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/google/uuid"
)

const (
	ClusterName     = "example_proxy_cluster"
	RouteName       = "local_route"
	UpstreamHost    = "www.envoyproxy.io"
	UpstreamPort    = 80
	Domains         = "*"
	Prefix          = "/"
	ListenerName    = "listener_0"
	ListenerPort    = 10000
	GrpcClusterName = "xds_cluster"
)

type XDSRequest struct {
	ClusterName     string
	UpstreamHost    string
	UpstreamPort    uint32
	RouteName       string
	Domains         []string
	Prefix          string
	ListenerName    string
	ListenerPort    uint32
	GrpcClusterName string
}

// cds????????????????????????????????????eds
func makeCluster(clusterName string, upstreamHost string, upstreamPort uint32) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_LOGICAL_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:       makeEndpoint(clusterName, upstreamHost, upstreamPort),
		DnsLookupFamily:      cluster.Cluster_V4_ONLY,
	}
}

// clusterName???eds?????????cds???name????????????????????????
func makeEndpoint(clusterName string, upstreamHost string, upstreamPort uint32) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									//Address:  UpstreamHost,
									Address: upstreamHost,
									PortSpecifier: &core.SocketAddress_PortValue{
										//PortValue: UpstreamPort,
										PortValue: upstreamPort,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func makeRoute(routeName string, domains []string, prefix string, clusterName string) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "local_service",
			Domains: domains,
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: prefix,
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: clusterName,
						},
						// ???????????????????????????host header???????????????????????????host_rewrite_literal????????????????????????????????????????????????ip?????????
						//HostRewriteSpecifier: &route.RouteAction_HostRewriteLiteral{
						//	HostRewriteLiteral: UpstreamHost,
						//},
					},
				},
			}},
		}},
	}
}

func makeHTTPListener(listenerName string, grpcClusterName string, route string, listenerPort uint32) *listener.Listener {
	routerConfig, _ := anypb.New(&router.Router{})
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			// ???rds????????????
			Rds: &hcm.Rds{
				// rds??????????????????????????????grpc??????????????????????????????????????????cds??????
				ConfigSource: makeConfigSource(grpcClusterName),
				// rds??????
				RouteConfigName: route,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name:       wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: routerConfig},
		}},
	}
	// ???????????????any?????????pb??????
	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: listenerPort,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

func makeConfigSource(grpcClusterName string) *core.ConfigSource {
	source := &core.ConfigSource{}
	// api??????????????????v3???
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		// api_config_source
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					// ????????????????????????grpc?????????????????????cds???name
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: grpcClusterName},
				},
			}},
		},
	}
	return source
}

// ??????Snapshot???????????????????????????????????????????????????????????????xDS?????????
func GenerateSnapshot(xDSRequest XDSRequest) *cache.Snapshot {
	uuid := uuid.New()
	// ????????????????????????????????????????????????????????????xDS???????????????????????????CDS???RDS???LDS
	// think: version ??????????????????????????????uuid??????
	snap, _ := cache.NewSnapshot(uuid.String(),
		map[resource.Type][]types.Resource{
			resource.ClusterType:  {makeCluster(xDSRequest.ClusterName, xDSRequest.UpstreamHost, xDSRequest.UpstreamPort)},
			resource.RouteType:    {makeRoute(xDSRequest.RouteName, xDSRequest.Domains, xDSRequest.Prefix, xDSRequest.ClusterName)},
			resource.ListenerType: {makeHTTPListener(xDSRequest.ListenerName, xDSRequest.GrpcClusterName, xDSRequest.RouteName, xDSRequest.ListenerPort)},
		},
	)
	return snap
}

// DefaultSnapshot ???????????????
func DefaultSnapshot() *cache.Snapshot {
	return GenerateSnapshot(XDSRequest{
		ClusterName:     ClusterName,
		UpstreamHost:    UpstreamHost,
		UpstreamPort:    UpstreamPort,
		RouteName:       RouteName,
		Domains:         []string{Domains},
		Prefix:          Prefix,
		ListenerName:    ListenerName,
		ListenerPort:    ListenerPort,
		GrpcClusterName: GrpcClusterName,
	})
}
