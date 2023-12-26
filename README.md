# etcd-registrar


protoc -I ./proto --go_opt=paths=source_relative --go_out=./proto/pb --go-grpc_opt=paths=source_relative --go-grpc_out=./proto/pb proto/*.proto