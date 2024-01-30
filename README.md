# etcd-registrar


protoc -I ./proto --go_opt=paths=source_relative --go_out=./proto/pb --go-grpc_opt=paths=source_relative --go-grpc_out=./proto/pb proto/*.proto

## TODO List
1. 负载均衡算法
2. 通过etcd进行低耦合的心跳检测。检测系统和被检测系统通过etcd上某个目录关联而非直接关联起来
3. 配置信息使用发布订阅，可能需要命令行工具
4. 取消代理，服务直接访问etcd