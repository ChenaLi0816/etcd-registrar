package main

type ProtoData struct {
	PackageName string
	PackagePath string
	Service     []Service
}

type Service struct {
	Name    string
	Methods []Method
}

func (s *Service) TypeName() string {
	return s.Name + "Server"
}

type Method struct {
	Name           string
	InputType      string
	OutputType     string
	IsClientStream bool
	IsServerStream bool
}
