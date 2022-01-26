package ui

import (
	"io/ioutil"

	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/widgets/mux"
	"github.com/hinshun/ptmux/ui/widgets/peerstyled"
	"github.com/sirupsen/logrus"
)

type UI struct {
	app    *gowid.App
	screen *screen
}

func New(id string) (*UI, error) {
	peerstyle := peerstyled.New(id, mux.New(id))

	s, err := newScreen(peerstyle)
	if err != nil {
		return nil, err
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	app, err := gowid.NewApp(gowid.AppArgs{
		Screen: s,
		View:   peerstyle,
		Log:    log,
		// EnableMouseMotion: true,
	})
	if err != nil {
		return nil, err
	}

	return &UI{
		app:    app,
		screen: s,
	}, nil
}

func (ui *UI) Screen() rvt.Screen {
	return ui.screen
}

func (ui *UI) Loop() {
	ui.app.MainLoop(gowid.UnhandledInputFunc(HandleQuitKeys))
}

func HandleQuitKeys(app gowid.IApp, event interface{}) bool {
	handled := false
	if ev, ok := event.(*tcell.EventKey); ok {
		if ev.Key() == tcell.KeyCtrlQ {
			app.Quit()
			handled = true
		}
	}
	return handled
}
