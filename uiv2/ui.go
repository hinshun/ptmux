package uiv2

import (
	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/uiv2/widgets/mux"
)

type UI struct {
	app *gowid.App
}

func New() (*UI, error) {
	app, err := gowid.NewApp(gowid.AppArgs{
		View: mux.New(),
	})
	if err != nil {
		return nil, err
	}

	return &UI{
		app: app,
	}, nil
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
