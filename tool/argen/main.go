package main

import "github.com/ChenaLi0816/etcd-registrar/tool/argen/args"

func main() {
	opt := &args.ArgOptions{}
	args.ParseArgs(opt)
	cmd, ok := opt.NewCmd()
	if !ok {
		return
	}
	info, err := cmd.CombinedOutput()
	if err != nil {
		panic(string(info))
	}
}
