package registrarclient

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	DEFAULT_LEASE_TIME = 10
)

type ClientOption interface {
	apply(*ClientOpts)
}

type ClientOpts struct {
	name      string
	file      bool
	localAddr string
	address   []string
	leaseTime int64
	passive   bool
}

func NewDefaultOptions() *ClientOpts {
	return &ClientOpts{
		name:      "",
		file:      false,
		localAddr: "",
		address:   []string{"127.0.0.1:8080"},
		leaseTime: DEFAULT_LEASE_TIME,
		passive:   false,
	}
}

func (opt *ClientOpts) ApplyOpts(option []ClientOption) {
	for _, op := range option {
		if _, ok := op.(*fileOption); ok && len(option) != 1 {
			panic("can't config by file and other options")
		}
		op.apply(opt)
	}
	if opt.name == "" || opt.localAddr == "" {
		panic("name or local address is null.")
	}
	if len(opt.address) == 0 {
		panic("registrar address is null")
	}
}

func (opt *ClientOpts) GetLeaseTime() int64 {
	return opt.leaseTime
}

func (opt *ClientOpts) GetAddress() []string {
	return opt.address
}

func (opt *ClientOpts) GetName() string {
	return opt.name
}

func (opt *ClientOpts) GetLocalAddr() string {
	return opt.localAddr
}

func (opt *ClientOpts) IsPassive() bool {
	return opt.passive
}

type fileOption struct {
	filepath string
}

func (op *fileOption) apply(opt *ClientOpts) {
	opt.file = true
	fp, err := os.Open(op.filepath)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = fp.Close()
	}()
	reader := bufio.NewReader(fp)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}
		if len(line) == 0 {
			continue
		}
		if line != "registrar address:" {
			panic(fmt.Sprintf("unrecgnized symbol: expect %s but %s", "registrar address:", line))
		}
		opt.address = append(opt.address, line)
		log.Println(line, "append success")
	}
}

func WithFileOption(filepath string) ClientOption {
	return &fileOption{
		filepath: filepath,
	}
}

type leaseTimeOption struct {
	leaseTime int64
}

func (op *leaseTimeOption) apply(opt *ClientOpts) {
	opt.leaseTime = op.leaseTime
}

func WithLeaseTime(t int64) ClientOption {
	return &leaseTimeOption{
		leaseTime: t,
	}
}

type addressOption struct {
	addr []string
}

func (op *addressOption) apply(opt *ClientOpts) {
	opt.address = op.addr
}

func WithRegistrarAddr(addr []string) ClientOption {
	return &addressOption{
		addr: addr,
	}
}

type serviceOption struct {
	name string
	addr string
}

func (op *serviceOption) apply(opt *ClientOpts) {
	opt.name = op.name
	opt.localAddr = op.addr
}

func WithService(name, addr string) ClientOption {
	return &serviceOption{
		name: name,
		addr: addr,
	}
}

type connOption struct {
	passive bool
}

func (op *connOption) apply(opt *ClientOpts) {
	opt.passive = op.passive
}

func WithPassive(b bool) ClientOption {
	return &connOption{passive: b}
}
