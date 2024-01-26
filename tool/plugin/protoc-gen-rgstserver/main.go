package main

import (
	"bytes"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func generateByPlugin(req *pluginpb.CodeGeneratorRequest) *pluginpb.CodeGeneratorResponse {
	opts := protogen.Options{}
	plugin, err := opts.New(req)
	if err != nil {
		return &pluginpb.CodeGeneratorResponse{
			Error: proto.String(err.Error()),
		}
	}
	for _, protoFile := range plugin.Files {
		data := new(tmplStruct)

		data.PbName = string(protoFile.GoPackageName)
		data.ServerpbName = string(protoFile.GoPackageName + "_server")
		data.PbPath = string(protoFile.GoImportPath)
		data.ServerpbPath = strings.TrimSuffix(data.PbPath, data.PbName) + data.ServerpbName

		data.Service = make([]service, 0, len(protoFile.Services))
		for _, svc := range protoFile.Services {
			data.Service = append(data.Service, service{ServiceName: svc.GoName})
		}

		filepath := strings.TrimSuffix(data.PbPath, "/"+data.PbName)
		filename := filepath + "/main.go"

		buf := bytes.NewBufferString("")
		if err = GenerateCode(data, buf); err != nil {
			return &pluginpb.CodeGeneratorResponse{
				Error: proto.String(err.Error()),
			}
		}

		content := buf.Bytes()
		_, err := plugin.NewGeneratedFile(filename, protogen.GoImportPath(filepath)).Write(content)
		if err != nil {
			return &pluginpb.CodeGeneratorResponse{
				Error: proto.String(err.Error()),
			}
		}
	}
	return plugin.Response()
}

func main() {
	// protoc 通过 stdin 传输数据。
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	// 解析收到的数据。
	var req pluginpb.CodeGeneratorRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		panic(err)
	}

	// 准备生成响应。

	// 调用生成逻辑（稍后定义）。
	resp := generateByPlugin(&req)

	// 将响应发送回 protoc。
	out, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		panic(err)
	}
}

func GenerateCode(data *tmplStruct, wr io.Writer) error {
	tmp, err := template.New("rgstserver").Parse(tmplString)
	if err != nil {
		return err
	}
	return tmp.Lookup("rgstserver").Execute(wr, data)
}
