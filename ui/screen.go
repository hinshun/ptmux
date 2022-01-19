package ui

import (
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/pkg/pubsub"
	"github.com/hinshun/ptmux/ui/widgets/peerstyled"
)

const (
	renderTopic = "render"
)

type screen struct {
	tcell.Screen
	pubsub    *pubsub.Pubsub
	peerstyle *peerstyled.Widget
}

func newScreen(peerstyle *peerstyled.Widget) (*screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	err = s.Init()
	if err != nil {
		return nil, err
	}
	s.EnableMouse()
	s.EnablePaste()

	ps := pubsub.New()
	return &screen{
		Screen:    s,
		pubsub:    ps,
		peerstyle: peerstyle,
	}, nil
}

func (s *screen) Show() {
	s.pubsub.Publish(renderTopic, "")
	s.Screen.Show()
}

func (s *screen) Sync() {
	s.pubsub.Publish(renderTopic, "")
	s.Screen.Sync()
}

func (s *screen) Clear() {
	s.pubsub.Publish(renderTopic, "")
	s.Screen.Clear()
}

func (s *screen) Subscribe(id string, ch chan string) {
	s.peerstyle.Add(id)
	s.pubsub.Subscribe(renderTopic, id, ch)
}

func (s *screen) Unsubscribe(id string) {
	s.peerstyle.Remove(id)
	s.pubsub.Unsubscribe(renderTopic, id)
}
