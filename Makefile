
build:
	go build -o protoc-gen-rgstserver.exe ./plugin/protoc-gen-rgstserver
	go build -o protoc-gen-implserver.exe ./plugin/protoc-gen-implserver

clean:
	go clean
	rm protoc-gen-rgstserver.exe
	rm protoc-gen-implserver.exe