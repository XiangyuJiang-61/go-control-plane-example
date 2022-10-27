# go-control-plane的案例
***
基于[go-control-plane](https://github.com/envoyproxy/go-control-plane)项目创建的一个案例，使用原有框架实现了通过一个gRPC client对控制平面进行热更新。
从而让控制平面分发通过client传入的最新配置给envoy
***
![img.png](img.png)
基本原理如上，通过一个gRPC客户端去更新控制平面里面的缓存，缓存更新的时候会通过订阅/发布去通知订阅了该控制中心的envoy更新配置

基本只能更新一部分xDS的对应字段，比如更新LDS的监听端口，更新RDS的路由的CDS名称，更新CDS内嵌套的EDS等

自己写着玩的，主要是用来作为理解动态配置控制平面的案例使用，肯定有更优的方法去处理

对这块的理解还不够深，如果有描述的问题还请谅解