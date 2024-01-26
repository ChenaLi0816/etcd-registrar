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
		for _, ext := range protoFile.Extensions {
			println(ext.GoIdent.String())
		}
		data := new(ProtoData)

		PbName := string(protoFile.GoPackageName)
		ServerpbName := string(protoFile.GoPackageName + "_server")
		PbPath := string(protoFile.GoImportPath)
		ServerpbPath := strings.TrimSuffix(PbPath, PbName) + ServerpbName

		data.PackagePath = PbPath
		data.PackageName = ServerpbName

		data.Service = make([]Service, 0, len(protoFile.Services))
		for _, service := range protoFile.Services {

			s := Service{}
			s.Name = service.GoName
			s.Methods = make([]Method, 0, len(service.Methods))
			for _, method := range service.Methods {
				m := Method{
					Name:           method.GoName,
					InputType:      method.Input.GoIdent.GoName,
					OutputType:     method.Output.GoIdent.GoName,
					IsClientStream: method.Desc.IsStreamingClient(),
					IsServerStream: method.Desc.IsStreamingServer(),
				}
				s.Methods = append(s.Methods, m)
			}
			data.Service = append(data.Service, s)
		}
		blo := strings.Split(protoFile.GeneratedFilenamePrefix, "/")

		filepath := ServerpbPath
		filename := filepath + "/" + blo[len(blo)-1] + "_server.go"

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

func GenerateCode(data *ProtoData, wr io.Writer) error {
	tmp, err := template.New("implserver").Parse(tmplString)
	if err != nil {
		return err
	}
	return tmp.Lookup("implserver").Execute(wr, data)
}
