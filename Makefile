
build:
	go build -o protoc-gen-rgstserver.exe ./tool/plugin/protoc-gen-rgstserver
	go build -o protoc-gen-implserver.exe ./tool/plugin/protoc-gen-implserver
	go build -o argen.exe ./tool/argen

clean:
	go clean
	rm protoc-gen-rgstserver.exe
	rm protoc-gen-implserver.exe
	rm argen.exe