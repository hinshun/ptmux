// Package styled provides a colored styled widget.

package styled

import (
	"fmt"

	"github.com/gcla/gowid"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
)

type Widget struct {
	gowid.IWidget
	*gowid.Callbacks
	gowid.SubWidgetCallbacks
	defaultID, lastID string
}

func New(defaultID string, inner gowid.IWidget) *Widget {
	return &Widget{
		IWidget:   inner,
		defaultID: defaultID,
	}
}

func (w *Widget) String() string {
	return fmt.Sprintf("styler[%v]", w.SubWidget())
}

func (w *Widget) SubWidget() gowid.IWidget {
	return w.IWidget
}

func (w *Widget) SetSubWidget(inner gowid.IWidget, app gowid.IApp) {
	w.IWidget = inner
	gowid.RunWidgetCallbacks(w, gowid.SubWidgetCB{}, app, w)
}

func (w *Widget) SubWidgetSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderSize {
	return w.RenderSize(size, focus, app)
}

func (w *Widget) RenderSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	return w.SubWidget().RenderSize(size, focus, app)
}

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	id := w.defaultID
	if evr, ok := ev.(*rvt.RemoteEvent); ok {
		id = evr.ID
	}
	w.lastID = id

	return gowid.UserInputIfSelectable(w.IWidget, ev, size, focus, app)
}

func (w *Widget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	var palette gowid.ICellStyler
	if p, ok := app.(wid.IP2PApp); ok {
		_, palette = p.FocusPalette(w.lastID)
	}

	canvas := w.SubWidget().Render(size, focus, app)
	if palette == nil {
		return canvas
	}

	var (
		f1 gowid.TCellColor
		b1 gowid.TCellColor
	)

	cols := canvas.BoxColumns()
	f, b, s := palette.GetStyle(app)
	for i := 0; i < cols*canvas.BoxRows(); i++ {
		col, row := i%cols, i/cols

		c := canvas.CellAt(col, row)
		c2 := c

		if f != nil {
			f1 = gowid.IColorToTCell(f, gowid.ColorNone, app.GetColorMode())
			c = c.WithForegroundColor(f1)
		}
		if b != nil {
			b1 = gowid.IColorToTCell(b, gowid.ColorNone, app.GetColorMode())
			c = c.WithBackgroundColor(b1)
		}

		c = c.WithStyle(s).MergeDisplayAttrsUnder(c2)
		canvas.SetCellAt(col, row, c)
	}

	return canvas
}
