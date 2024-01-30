package main

import (
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/tool/argen/args"
	"os"
)

func main() {
	opt := &args.ArgOptions{}
	args.ParseArgs(opt)
	cmd, ok := opt.NewCmd()
	if !ok {
		return
	}
	info, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(info))
		os.Exit(1)
	}
}
