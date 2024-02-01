ifeq ($(OS),Windows_NT)
    detected_OS := Windows
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        detected_OS := Linux
    endif
    ifeq ($(UNAME_S),Darwin)
        detected_OS := MacOS
    endif
endif

test:
    @echo Detected OS: $(detected_OS)

build:
	go build -o protoc-gen-rgstserver.exe ./tool/plugin/protoc-gen-rgstserver
	go build -o protoc-gen-implserver.exe ./tool/plugin/protoc-gen-implserver
	go build -o argen.exe ./tool/argen

server:
	go build -o runserver.exe ./server

clean:
	go clean
	rm *.exe