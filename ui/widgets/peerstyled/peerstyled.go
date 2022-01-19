package peerstyled

import (
	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
)

var (
	Stylers = []gowid.ICellStyler{
		gowid.MakeForeground(gowid.ColorBlue),
		gowid.MakeForeground(gowid.ColorGreen),
	}
)

type Widget struct {
	gowid.IWidget
	paletteIdx   int
	palette      map[string]gowid.ICellStyler
	clickTargets map[string]gowid.ClickTargets
	lastMouse    map[string]gowid.MouseState
}

func New(inner gowid.IWidget) *Widget {
	w := &Widget{
		IWidget:      inner,
		palette:      make(map[string]gowid.ICellStyler),
		clickTargets: make(map[string]gowid.ClickTargets),
		lastMouse:    make(map[string]gowid.MouseState),
	}
	w.Add(wid.DefaultID)
	return w
}

func (w *Widget) Add(id string) {
	w.palette[id] = Stylers[w.paletteIdx]
	w.paletteIdx++
	w.clickTargets[id] = gowid.MakeClickTargets()
	w.lastMouse[id] = gowid.MouseState{}
}

func (w *Widget) Remove(id string) {
	delete(w.palette, id)
	delete(w.clickTargets, id)
	delete(w.lastMouse, id)
}

func (w *Widget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return w.IWidget.Render(size, focus, wid.WithP2PContext(app, w.palette, gowid.ClickTargets{}, gowid.MouseState{}, gowid.MouseState{}))
}

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	id := wid.DefaultID
	evt := ev
	if evr, ok := ev.(*rvt.RemoteEvent); ok {
		id = evr.ID
		evt = evr.Event
	}

	if evm, ok := evt.(*tcell.EventMouse); ok {
		mouseState := gowid.MouseState{}
		switch evm.Buttons() {
		case tcell.Button1:
			mouseState.MouseLeftClicked = true
		case tcell.Button2:
			mouseState.MouseMiddleClicked = true
		case tcell.Button3:
			mouseState.MouseRightClicked = true
		}

		app = wid.WithP2PContext(app, w.palette, w.clickTargets[id], mouseState, w.lastMouse[id])
		handled := w.IWidget.UserInput(ev, size, focus, app)

		if evm.Buttons() == tcell.ButtonNone {
			w.clickTargets[id].DeleteClickTargets(tcell.Button1)
			w.clickTargets[id].DeleteClickTargets(tcell.Button2)
			w.clickTargets[id].DeleteClickTargets(tcell.Button3)
		}

		w.lastMouse[id] = mouseState
		return handled
	}

	app = wid.WithP2PContext(app, w.palette, w.clickTargets[id], gowid.MouseState{}, w.lastMouse[id])
	return w.IWidget.UserInput(ev, size, focus, app)
}
