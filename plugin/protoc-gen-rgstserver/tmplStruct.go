package main

type tmplStruct struct {
	PbPath       string
	ServerpbPath string
	PbName       string
	ServerpbName string
	Service      []service
}

type service struct {
	ServiceName string
}
