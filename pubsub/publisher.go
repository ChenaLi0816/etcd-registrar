package pubsub

import (
	"errors"
	"github.com/google/uuid"
	"sync"
)

const (
	BUFSIZE = 10
)

type Publisher struct {
	ch sync.Map
}

var manager *Publisher = nil

func NewPublisher() *Publisher {
	if manager == nil {
		manager = &Publisher{
			ch: sync.Map{},
		}
	}
	return manager
}

func (m *Publisher) Close() {
	m.ch.Range(func(key, value any) bool {
		close(value.(*channel).ch)
		return true
	})
	m.ch = sync.Map{}
}

func (m *Publisher) CloseChannel(name string) error {
	c, ok := m.ch.Load(name)
	if !ok {
		return errors.New(name + "don't exist")
	}
	close(c.(*channel).ch)
	m.ch.Delete(name)
	return nil
}

func (m *Publisher) Publish(name string, msg interface{}) {
	var c *channel
	ch, ok := m.ch.Load(name)
	if !ok {
		c = newChannel(name)
	} else {
		c = ch.(*channel)
	}
	c.ch <- msg
}

func (m *Publisher) AddSubscriber(s Subscriber, name string) string {
	var c *channel
	ch, ok := m.ch.Load(name)
	if !ok {
		c = newChannel(name)
	} else {
		c = ch.(*channel)
	}
	id := uuid.NewString()
	c.subscribers.Store(id, s)
	return id
}

func (m *Publisher) RemoveSubscriber(id string, name string) {
	var c *channel
	ch, ok := m.ch.Load(name)
	if !ok {
		return
	} else {
		c = ch.(*channel)
	}
	c.subscribers.Delete(id)
}

type channel struct {
	ch          chan interface{}
	subscribers sync.Map
}

func newChannel(name string) *channel {
	m := NewPublisher()
	c := &channel{
		ch:          make(chan interface{}, BUFSIZE),
		subscribers: sync.Map{},
	}
	m.ch.Store(name, c)
	go c.Recv()
	return c
}

func (c *channel) Recv() {
	for {
		select {
		case msg, ok := <-c.ch:
			if !ok {
				return
			}
			c.subscribers.Range(func(key, value any) bool {
				value.(Subscriber).RecvMsg(msg)
				return true
			})
		}
	}
}

func lenOfSyncMap(mp *sync.Map) int {
	num := 0
	mp.Range(func(key, value any) bool {
		num++
		return true
	})
	return num
}
