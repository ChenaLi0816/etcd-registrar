package registrarserver

const (
	chanSize = 10
	maxRetry = 3
)

type configSubscriber struct {
	name       string
	retryTimes int
}

func (s *configSubscriber) RecvMsg(msg interface{}) {
	c, _ := etcdServer.msgChan.Load(s.name)
	ch := c.(chan string)
	select {
	case ch <- msg.(string):
	default:
		s.retryTimes++
		if s.retryTimes > maxRetry {
			publisher.RemoveSubscriber(s.name, "config")
		}
	}
}

func (s *configSubscriber) BeRemoved() {
	etcdServer.msgChan.Delete(s.name)
}

func newConfigSubscriber(name string) *configSubscriber {
	etcdServer.msgChan.Store(name, make(chan string, chanSize))
	return &configSubscriber{name: name}
}
