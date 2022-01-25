// Package pile provides a widget for organizing other widgets in a vertical stack.
package pile

import (
	"fmt"
	"strings"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
)

type IWidget interface {
	gowid.IWidget
	gowid.ICompositeMultipleDimensions
	gowid.ISettableDimensions
	gowid.ISettableSubWidgets
	gowid.IIdentity
	wid.IFocus
	SubWidgetSize(size gowid.IRenderSize, val int, sub gowid.IWidget, dim gowid.IWidgetDimension) gowid.IRenderSize
	RenderSubWidgets(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.ICanvas
	RenderedSubWidgetsSizes(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.IRenderBox
	RenderBoxMaker(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp, fn BoxMaker) ([]gowid.IRenderBox, []gowid.IRenderSize)
}

type Widget struct {
	gowid.AddressProvidesID
	widgets   []gowid.IContainerWidget
	focus     map[string]int
	defaultID string
}

func New(defaultID string, widgets []gowid.IContainerWidget) *Widget {
	return &Widget{
		widgets:   widgets,
		focus:     make(map[string]int),
		defaultID: defaultID,
	}
}

func (w *Widget) String() string {
	rows := make([]string, len(w.widgets))
	for i := 0; i < len(rows); i++ {
		rows[i] = fmt.Sprintf("%v", w.widgets[i])
	}
	return fmt.Sprintf("pile[%s]", strings.Join(rows, ","))
}

func (w *Widget) SetFocus(id string, i int) {
	w.focus[id] = i
}

func (w *Widget) Focus(id string) int {
	focus, ok := w.focus[id]
	if !ok {
		return 1
	}
	return focus
}

func (w *Widget) ReverseFocus() [][]string {
	rfocus := make([][]string, len(w.widgets))
	for id, focus := range w.focus {
		rfocus[focus] = append(rfocus[focus], id)
	}
	return rfocus
}

func (w *Widget) SubWidgets() []gowid.IWidget {
	res := make([]gowid.IWidget, len(w.widgets))
	for i, iw := range w.widgets {
		res[i] = iw
	}
	return res
}

func (w *Widget) SetSubWidgets(widgets []gowid.IWidget, app gowid.IApp) {
	ws := make([]gowid.IContainerWidget, len(widgets))
	for i, iw := range widgets {
		if iwc, ok := iw.(gowid.IContainerWidget); ok {
			ws[i] = iwc
		} else {
			ws[i] = &gowid.ContainerWidget{IWidget: iw, D: gowid.RenderFlow{}}
		}
	}
	w.widgets = ws
}

func (w *Widget) Dimensions() []gowid.IWidgetDimension {
	res := make([]gowid.IWidgetDimension, len(w.widgets))
	for i, iw := range w.widgets {
		res[i] = iw.Dimension()
	}
	return res
}

func (w *Widget) SetDimensions(dimensions []gowid.IWidgetDimension, app gowid.IApp) {
	for i, id := range dimensions {
		w.widgets[i].SetDimension(id)
	}
}

func (w *Widget) Selectable() bool {
	return gowid.SelectableIfAnySubWidgetsAre(w)
}

func (w *Widget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return Render(w, size, focus, app)
}

func (w *Widget) RenderedSubWidgetsSizes(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.IRenderBox {
	res, _ := RenderedChildrenSizes(w, size, focus, app)
	return res
}

func (w *Widget) RenderSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	return RenderSize(w, size, focus, app)
}

func (w *Widget) RenderSubWidgets(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.ICanvas {
	return RenderSubwidgets(w, size, focus, app)
}

func (w *Widget) RenderBoxMaker(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp, fn BoxMaker) ([]gowid.IRenderBox, []gowid.IRenderSize) {
	return RenderBoxMaker(w, size, focus, app, fn)
}

// SubWidgetSize is the size that should be used to render a child widget, based on the size used to render the parent.
func (w *Widget) SubWidgetSize(size gowid.IRenderSize, newY int, sub gowid.IWidget, dim gowid.IWidgetDimension) gowid.IRenderSize {
	return gowid.ComputeVerticalSubSizeUnsafe(size, dim, -1, newY)
}

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	handled := false
	subs := w.SubWidgets()

	// An array of IRenderBoxes
	ss, ss2 := RenderedChildrenSizes(w, size, focus, app)

	evt := ev
	id := w.defaultID
	if evr, ok := ev.(*rvt.RemoteEvent); ok {
		evt = evr.Event
		id = evr.ID
	}

	rfocus := w.ReverseFocus()
	subfocus := w.Focus(id)
	if evm, ok := evt.(*tcell.EventMouse); ok {
		// A left click sets focus if the widget is selectable and would take the mouse input
		_, my := evm.Position()
		curY := 0
		for i, c := range ss {
			curY += c.BoxRows()
			if my >= curY || my < curY-c.BoxRows() {
				continue
			}

			subSize := ss2[i]
			handled = subs[i].UserInput(gowid.TranslatedMouseEvent(ev, 0, -curY+c.BoxRows()), subSize, focus, wid.WithFocus(app, rfocus[i]))

			switch evm.Buttons() {
			case tcell.Button1, tcell.Button2, tcell.Button3:
				app.SetClickTarget(evm.Buttons(), w)
			case tcell.ButtonNone:
				if !app.GetLastMouseState().NoButtonClicked() && subs[i].Selectable() {
					clicked := false
					app.ClickTarget(func(_ tcell.ButtonMask, iw gowid.IIdentityWidget) {
						if iw != nil && iw.ID() == w.ID() {
							clicked = true
						}
					})
					if clicked {
						w.SetFocus(id, i)
					}
				}
			}
			break
		}
	} else {
		srows := 0
		for i := 0; i < subfocus; i++ {
			srows += ss[i].BoxRows()
		}
		subSize := ss2[subfocus]
		handled = gowid.UserInputIfSelectable(subs[subfocus], gowid.TranslatedMouseEvent(ev, 0, -srows), subSize, focus, wid.WithFocus(app, rfocus[subfocus]))
	}

	return handled
}

func RenderSize(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	sizes := w.RenderedSubWidgetsSizes(size, focus, app)

	maxcol := 0
	maxrow := 0

	for _, sz := range sizes {
		maxcol = gwutil.Max(maxcol, sz.BoxColumns())
		maxrow += sz.BoxRows()
	}

	if sz, ok := size.(gowid.IRenderBox); ok {
		maxrow = gwutil.Min(maxrow, sz.BoxRows())
	}

	return gowid.RenderBox{maxcol, maxrow}
}

func Render(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	canvases := w.RenderSubWidgets(size, focus, app)

	rows, ok := size.(gowid.IRows)
	haveMaxRow := ok

	res := gowid.NewCanvas()
	trim := false
	for i := 0; i < len(canvases); i++ {
		res.AppendBelow(canvases[i], false, false)
		if haveMaxRow && res.BoxRows() >= rows.Rows() {
			trim = true
			break
		}
	}

	if trim {
		res.Truncate(0, res.BoxRows()-rows.Rows())
	}
	if haveMaxRow && res.BoxRows() < rows.Rows() {
		gowid.AppendBlankLines(res, rows.Rows()-res.BoxRows())
	}
	if cols, ok := size.(gowid.IColumns); ok {
		res.ExtendRight(gowid.EmptyLine(cols.Columns() - res.BoxColumns()))
	}

	return res
}

func RenderedChildrenSizes(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) ([]gowid.IRenderBox, []gowid.IRenderSize) {
	return w.RenderBoxMaker(size, focus, app, func(w gowid.IWidget, subSize gowid.IRenderSize, focus gowid.Selector, i int, subApp gowid.IApp) gowid.IRenderBox {
		return w.RenderSize(subSize, focus, subApp)
	})
}

func RenderSubwidgets(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.ICanvas {
	rfocus := w.ReverseFocus()
	boxes, _ := w.RenderBoxMaker(size, focus, app, func(w gowid.IWidget, subSize gowid.IRenderSize, focus gowid.Selector, i int, app gowid.IApp) gowid.IRenderBox {
		return w.Render(subSize, focus, wid.WithFocus(app, rfocus[i]))
	})

	canvases := make([]gowid.ICanvas, len(boxes))
	for i := 0; i < len(boxes); i++ {
		canvases[i] = boxes[i].(gowid.ICanvas)
	}
	return canvases
}

type BoxMaker func(gowid.IWidget, gowid.IRenderSize, gowid.Selector, int, gowid.IApp) gowid.IRenderBox

func RenderBoxMaker(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp, fn BoxMaker) ([]gowid.IRenderBox, []gowid.IRenderSize) {
	// Initialize with invalid weight to distinguish between what is set is what isn't.
	subs := w.SubWidgets()
	wlen := len(subs)
	heights := make([]int, wlen)
	for i := 0; i < wlen; i++ {
		heights[i] = -1
	}

	totalWeight := 0
	dims := w.Dimensions()
	for i := 0; i < wlen; i++ {
		rw := dims[i].(gowid.IRenderWithWeight)
		totalWeight += rw.Weight()
	}

	// Track the last height row, to adjust for floating point errors.
	lastHeightRow := -1

	box := size.(gowid.IRenderBox)
	rowsToDivideUp := box.BoxRows()
	rowsLeft := rowsToDivideUp
	for rowsLeft > 0 {
		finished := true
		rowsToDivideUp = rowsLeft
		for i := 0; i < wlen; i++ {
			rw := dims[i].(gowid.IRenderWithWeight)
			rows := int(((float32(rw.Weight()) / float32(totalWeight)) * float32(rowsToDivideUp)) + 0.5)
			if rows > rowsLeft {
				rows = rowsLeft
			}
			if rows > 0 {
				if heights[i] == -1 {
					heights[i] = 0
				}
				heights[i] += rows
				rowsLeft -= rows
				lastHeightRow = i
				finished = false
			}
		}
		if finished {
			break
		}
	}
	if lastHeightRow != -1 && rowsLeft > 0 {
		heights[lastHeightRow] += rowsLeft
	}

	// Finally render
	res := make([]gowid.IRenderBox, wlen)
	resSS := make([]gowid.IRenderSize, wlen)
	for i := 0; i < wlen; i++ {
		ss := gowid.RenderBox{box.BoxColumns(), heights[i]}
		res[i] = fn(subs[i], ss, focus, i, app)
		resSS[i] = ss
	}

	return res, resSS
}
