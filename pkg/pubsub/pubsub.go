package pubsub

import "sync"

type Pubsub struct {
	mu     sync.RWMutex
	subs   map[string]map[string]chan string
	closed bool
}

func New() *Pubsub {
	return &Pubsub{
		subs: make(map[string]map[string]chan string),
	}
}

func (ps *Pubsub) Subscribe(topic, id string, ch chan string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	topicSubs, ok := ps.subs[topic]
	if !ok {
		topicSubs = make(map[string]chan string)
	}

	sub, ok := topicSubs[id]
	if ok {
		close(sub)
		delete(topicSubs, id)
	}

	topicSubs[id] = ch
	ps.subs[topic] = topicSubs
}

func (ps *Pubsub) Unsubscribe(topic, id string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return
	}

	topicSubs, ok := ps.subs[topic]
	if !ok {
		return
	}

	ch, ok := topicSubs[id]
	if ok {
		close(ch)
		delete(topicSubs, id)
	}
}

func (ps *Pubsub) Publish(topic string, msg string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return
	}

	for _, ch := range ps.subs[topic] {
		ch <- msg
	}
}

func (ps *Pubsub) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.closed {
		ps.closed = true
		for _, subs := range ps.subs {
			for _, ch := range subs {
				close(ch)
			}
		}
	}

	return nil
}
