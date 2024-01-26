package args

import (
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	BASIC = iota
	GRPC
	IMPL
	RGST
)

type Parser interface {
	parseArgs()
	NewCmd() (*exec.Cmd, bool)
}

type ArgOptions struct {
	module       string
	protoPath    []string
	includePath  []string
	generatePath string `default:"."`
	mode         int
	hasParsed    bool
	infoFlag     bool
}

func (a *ArgOptions) parseArgs() {
	arg := os.Args[1:]
	i := 0
	for i < len(arg) {
		switch arg[i] {
		case "-I":
			fallthrough
		case "--proto_path":
			i++
			a.includePath = append(a.includePath, arg[i])
		case "-g":
			fallthrough
		case "--grpc":
			if a.mode < GRPC {
				a.mode = GRPC
			}
		case "-i":
			fallthrough
		case "--implement":
			if a.mode < IMPL {
				a.mode = IMPL
			}
		case "-r":
			fallthrough
		case "--register":
			if a.mode < RGST {
				a.mode = RGST
			}
		case "-v":
			fallthrough
		case "--version":
			fmt.Println("argen version:", versionInfo)
			a.infoFlag = true
			return
		case "-h":
			fallthrough
		case "--help":
			fmt.Println(helpInfo)
			a.infoFlag = true
			return
		default:
			if strings.HasSuffix(arg[i], ".proto") {
				a.protoPath = arg[i:]
				break
			}
			fmt.Println("err: unrecognized flag", arg[i])
			os.Exit(0)
		}
		i++
	}
	goPath := utils.GetGOPATH()
	filepath.Join(goPath, "src")
	pwd, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}
	moduleName, path, found := utils.SearchGoMod(pwd)
	if !found {
		panic("please init go mod first")
	}

	a.module = moduleName
	a.generatePath = path

	if a.protoPath == nil {
		fmt.Println("err: no protos, please check")
		os.Exit(0)
	}
	if a.module == "" {
		fmt.Println("err: module is null, please check")
		os.Exit(0)
	}
	a.hasParsed = true
}

func (a *ArgOptions) NewCmd() (*exec.Cmd, bool) {
	if a.infoFlag {
		return nil, false
	}
	if !a.hasParsed {
		panic("please parse args first")
	}
	cmdArgs := make([]string, 0, 20)
	for _, path := range a.includePath {
		cmdArgs = append(cmdArgs, "-I", path)
	}
	switch a.mode {
	case RGST:
		cmdArgs = a.optRgst(cmdArgs)
		fallthrough
	case IMPL:
		cmdArgs = a.optImpl(cmdArgs)
		fallthrough
	case GRPC:
		cmdArgs = a.optGrpc(cmdArgs)
		fallthrough
	case BASIC:
		cmdArgs = a.optBasic(cmdArgs)
	}
	cmdArgs = append(cmdArgs, a.protoPath...)

	return exec.Command("protoc", cmdArgs...), true
}

func (a *ArgOptions) optBasic(args []string) []string {
	args = append(args, "--go_out="+a.generatePath, "--go_opt=module="+a.module)
	return args
}

func (a *ArgOptions) optGrpc(args []string) []string {
	args = append(args, "--go-grpc_out="+a.generatePath, "--go-grpc_opt=module="+a.module)
	return args
}

func (a *ArgOptions) optImpl(args []string) []string {
	// TODO .exe
	args = append(args, "--plugin=protoc-gen-implserver.exe", "--implserver_opt=module="+a.module, "--implserver_out="+a.generatePath)
	return args
}

func (a *ArgOptions) optRgst(args []string) []string {
	// TODO .exe
	args = append(args, "--plugin=protoc-gen-rgstserver.exe", "--rgstserver_opt=module="+a.module, "--rgstserver_out="+a.generatePath)
	return args
}

func ParseArgs(p Parser) Parser {
	p.parseArgs()
	return p
}
