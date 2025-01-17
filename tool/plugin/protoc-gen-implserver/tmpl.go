package main

const tmplString = `package {{.PackageName}}

import (
    "context"
    pb "{{.PackagePath}}"
)
{{range .Service}}{{$serviceName := .Name}}{{$typeName := .TypeName}}
type {{$typeName}} struct {
    pb.Unimplemented{{.Name}}Server
}
{{range .Methods}}{{if .IsClientStream}}{{if .IsServerStream}}
func (s * {{$typeName}}) {{.Name}}(stream pb.{{$serviceName}}_{{.Name}}Server) error {
    // TODO your code here...
    return nil
}
{{end}}{{else}}{{if .IsServerStream}}
func (s * {{$typeName}}) {{.Name}}(in *pb.{{.InputType}}, stream pb.{{$serviceName}}_{{.Name}}Server) error {
    // TODO your code here...
    return nil
}
{{else}}
func (s * {{$typeName}}) {{.Name}}(ctx context.Context, input *pb.{{.InputType}}) (*pb.{{.OutputType}}, error) {
    // TODO your code here...
    return nil, nil
}
{{end}}{{end}}{{end}}
func New{{$typeName}}() *{{$typeName}} {
    return &{{$typeName}}{}
}
{{end}}`
