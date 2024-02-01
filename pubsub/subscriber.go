package pubsub

import "fmt"

type Subscriber interface {
	RecvMsg(interface{})
	BeRemoved()
}

type SubscriberEvent struct {
	stopChan chan struct{}
}

func (s *SubscriberEvent) RecvMsg(msg interface{}) {
	fmt.Printf("subscriber recv msg %v\n", msg)
}
