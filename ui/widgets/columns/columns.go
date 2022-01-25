// Package columns provides a widget for organizing other widgets in columns.
package columns

import (
	"fmt"
	"os"
	"strings"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	"github.com/gcla/gowid/widgets/fill"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
)

var log *os.File

func init() {
	log, _ = os.Create("columns.log")
}

type IWidget interface {
	gowid.IWidget
	gowid.ICompositeMultipleDimensions
	gowid.ISettableDimensions
	gowid.ISettableSubWidgets
	gowid.IIdentity
	wid.IFocus
	SubWidgetSize(size gowid.IRenderSize, val int, sub gowid.IWidget, dim gowid.IWidgetDimension) gowid.IRenderSize
	RenderSubWidgets(size gowid.IRenderSize, focus gowid.Selector, rfocus [][]string, app gowid.IApp) []gowid.ICanvas
	RenderedSubWidgetsSizes(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.IRenderBox
	WidgetWidths(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []int
}

type Widget struct {
	gowid.AddressProvidesID
	widgets      []gowid.IContainerWidget
	focus        map[string]int
	widthHelper  []bool // optimizations to save frequent array allocations during use
	widthHelper2 []bool
	defaultID    string
}

func New(defaultID string, widgets []gowid.IContainerWidget) *Widget {
	return &Widget{
		widgets:      widgets,
		focus:        make(map[string]int),
		widthHelper:  make([]bool, len(widgets)),
		widthHelper2: make([]bool, len(widgets)),
		defaultID:    defaultID,
	}
}

func (w *Widget) String() string {
	cols := make([]string, len(w.widgets))
	for i := 0; i < len(cols); i++ {
		cols[i] = fmt.Sprintf("%v", w.widgets[i])
	}
	return fmt.Sprintf("columns[%s]", strings.Join(cols, ","))
}

func (w *Widget) SetFocus(id string, i int) {
	w.focus[id] = i
}

func (w *Widget) Focus(id string) int {
	focus, ok := w.focus[id]
	if !ok {
		w.focus[id] = 1
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
	w.widthHelper = make([]bool, len(widgets))
	w.widthHelper2 = make([]bool, len(widgets))
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

// RenderSize computes the size of this widget when it renders. This is
// done by computing the sizes of each subwidget, then arranging them the
// same way that Render() does.
func (w *Widget) RenderSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	return RenderSize(w, size, focus, app)
}

func (w *Widget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return Render(w, size, focus, app)
}

func (w *Widget) RenderSubWidgets(size gowid.IRenderSize, focus gowid.Selector, rfocus [][]string, app gowid.IApp) []gowid.ICanvas {
	return RenderSubWidgets(w, size, focus, rfocus, app)
}

func (w *Widget) RenderedSubWidgetsSizes(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.IRenderBox {
	return RenderedSubWidgetsSizes(w, size, focus, app)
}

// Return a slice of ints representing the width in columns for each of the subwidgets to be rendered
// in this context given the size argument.
func (w *Widget) WidgetWidths(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []int {
	return WidgetWidths(w, size, focus, app)
}

// Construct the context in which each subwidget will be rendered. It's important to
// preserve the type of context e.g. a subwidget may only support being rendered in a
// fixed context. The newX parameter is the width the subwidget will have within the
// context of the Columns widget.
func (w *Widget) SubWidgetSize(size gowid.IRenderSize, newX int, sub gowid.IWidget, dim gowid.IWidgetDimension) gowid.IRenderSize {
	return SubWidgetSize(size, newX, dim)
}

type IWidthHelper interface {
	WidthHelpers() ([]bool, []bool)
}

var _ IWidthHelper = (*Widget)(nil)

func (w *Widget) WidthHelpers() ([]bool, []bool) {
	return w.widthHelper, w.widthHelper2
}

func SubWidgetSize(size gowid.IRenderSize, newX int, dim gowid.IWidgetDimension) gowid.IRenderSize {
	var subSize gowid.IRenderSize

	switch sz := size.(type) {
	case gowid.IRenderFixed:
		switch dim.(type) {
		case gowid.IRenderBox:
			subSize = dim
		default:
			subSize = gowid.RenderFixed{}
		}
	case gowid.IRenderBox:
		switch dim.(type) {
		case gowid.IRenderFixed:
			subSize = gowid.RenderFixed{}
		case gowid.IRenderFlow:
			subSize = gowid.RenderFlowWith{C: newX}
		case gowid.IRenderWithUnits, gowid.IRenderWithWeight:
			subSize = gowid.RenderBox{C: newX, R: sz.BoxRows()}
		default:
			subSize = gowid.RenderBox{C: newX, R: sz.BoxRows()}
		}
	case gowid.IRenderFlowWith:
		switch dim.(type) {
		case gowid.IRenderFixed:
			subSize = gowid.RenderFixed{}
		case gowid.IRenderFlow, gowid.IRenderWithUnits, gowid.IRenderWithWeight, gowid.IRenderRelative:
			// The newX argument is already computed to be the right number of cols for the subwidget
			subSize = gowid.RenderFlowWith{C: newX}
		default:
			panic(gowid.DimensionError{Size: size, Dim: dim})
		}
	default:
		panic(gowid.DimensionError{Size: size, Dim: dim})
	}
	return subSize
}

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	handled := false
	subSizes := w.WidgetWidths(size, focus, app)

	dims := w.Dimensions()
	subs := w.SubWidgets()

	evt := ev
	id := w.defaultID
	if evr, ok := ev.(*rvt.RemoteEvent); ok {
		evt = evr.Event
		id = evr.ID
	}

	rfocus := w.ReverseFocus()
	subfocus := w.Focus(id)
	if evm, ok := evt.(*tcell.EventMouse); ok {
		log.Write([]byte(fmt.Sprintf("mouse: %v\n", ev)))
		curX := 0
		mx, _ := evm.Position()
		for i, c := range subSizes {
			curX += c
			if mx >= curX || mx < curX-c {
				continue
			}

			subSize := w.SubWidgetSize(size, c, subs[i], dims[i])
			handled = subs[i].UserInput(gowid.TranslatedMouseEvent(ev, -curX+c, 0), subSize, focus, wid.WithFocus(app, rfocus[i]))

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
		subC := subSizes[subfocus] // guaranteed to be a box
		subSize := w.SubWidgetSize(size, subC, subs[subfocus], dims[subfocus])
		handled = gowid.UserInputIfSelectable(subs[subfocus], ev, subSize, focus, wid.WithFocus(app, rfocus[subfocus]))
	}

	return handled
}

func WidgetWidths(w gowid.ICompositeMultipleDimensions, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []int {
	return widgetWidthsExt(w, w.SubWidgets(), w.Dimensions(), size, focus, app)
}

// Precompute dims and subs
func widgetWidthsExt(w interface{}, subs []gowid.IWidget, dims []gowid.IWidgetDimension, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []int {
	lenw := len(subs)

	res := make([]int, lenw)
	var widthHelper []bool
	var widthHelper2 []bool
	if w, ok := w.(IWidthHelper); ok {
		// Save some allocations
		widthHelper, widthHelper2 = w.WidthHelpers()
		defer func() {
			for i := 0; i < len(widthHelper); i++ {
				widthHelper[i] = false
				widthHelper2[i] = false
			}
		}()
	} else {
		widthHelper = make([]bool, lenw)
		widthHelper2 = make([]bool, lenw)
	}

	haveColsTotal := false
	var colsTotal int
	if _, ok := size.(gowid.IRenderFixed); !ok {
		cols, ok := size.(gowid.IColumns)
		if !ok {
			panic(gowid.WidgetSizeError{Widget: w, Size: size, Required: "gowid.IColumns"})
		}
		colsTotal = cols.Columns()
		haveColsTotal = true
	}

	colsUsed := 0
	totalWeight := 0

	// First, render the widgets whose width is known
	for i := 0; i < lenw; i++ {
		// This doesn't support IRenderFlow. That type comes with an associated width e.g.
		// "Flow with 25 columns". We don't have any way to apportion those columns amongst
		// the overall width for the widget.
		switch w2 := dims[i].(type) {
		case gowid.IRenderWithWeight:
			// widget must be weighted
			totalWeight += w2.Weight()
			widthHelper[i] = false
			widthHelper2[i] = false
		default:
			panic(gowid.DimensionError{Size: size, Dim: w2})
		}
	}

	var colsLeft int
	var colsToDivideUp int
	if haveColsTotal {
		colsToDivideUp = colsTotal - colsUsed
		colsLeft = colsToDivideUp
	}

	// Now, divide up the remaining space among the weight columns
	lasti := -1
	maxedOut := false
	for {
		if colsLeft == 0 {
			break
		}
		doneone := false
		totalWeight = 0
		for i := 0; i < lenw; i++ {
			if w2, ok := dims[i].(gowid.IRenderWithWeight); ok && !widthHelper[i] {
				totalWeight += w2.Weight()
			}
		}
		colsToDivideUp = colsLeft
		for i := 0; i < lenw; i++ {
			// Can only be weight here if !helper[i] ; but not sufficient for it to be eligible
			if !widthHelper[i] {
				cols := int(((float32(dims[i].(gowid.IRenderWithWeight).Weight()) / float32(totalWeight)) * float32(colsToDivideUp)) + 0.5)
				if !maxedOut {
					if max, ok := dims[i].(gowid.IRenderMaxUnits); ok {
						if cols >= max.MaxUnits() {
							cols = max.MaxUnits()
							widthHelper[i] = true // this one is done
						}
					}
				}
				if cols > colsLeft {
					cols = colsLeft
				}
				if cols > 0 {
					if res[i] == -1 {
						res[i] = 0
					}
					res[i] += cols
					colsLeft -= cols
					lasti = gwutil.Max(i, lasti)
					doneone = true
				}
			}
		}
		if !doneone {
			// We used up all our extra space, after all weighted columns were maxed out. So
			// we're done. Any extra space (should be just 1 unit at most) goes on the last columns.
			if maxedOut {
				break
			}
			// All the weighted columns have been assigned, and all were maxed out. We still
			// have space to assign. So now grow the weighted columns, even though they don't need
			// any more space for a full render - what else to do with the space!
			maxedOut = true
			// Reset; all false indices will be indices of weighted columns again
			for i := 0; i < len(widthHelper); i++ {
				widthHelper[i] = widthHelper2[i]
			}
		}
	}
	if lasti != -1 && colsLeft > 0 {
		res[lasti] += colsLeft
	}

	return res
}

func RenderSize(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	sizes := w.RenderedSubWidgetsSizes(size, focus, app)

	maxcol, maxrow := 0, 0

	for _, sz := range sizes {
		maxcol += sz.BoxColumns()
		maxrow = gwutil.Max(maxrow, sz.BoxRows())
	}

	if cols, ok := size.(gowid.IColumns); ok {
		maxcol = cols.Columns()
		if rows, ok2 := size.(gowid.IRows); ok2 {
			maxrow = rows.Rows()
		}
	}

	return gowid.RenderBox{maxcol, maxrow}
}

func Render(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	rfocus := w.ReverseFocus()
	canvases := w.RenderSubWidgets(size, focus, rfocus, app)

	// Assemble subcanvases into final canvas
	subs := w.SubWidgets()
	canvas := gowid.NewCanvas()
	for i := 0; i < len(subs); i++ {
		diff := canvas.BoxRows() - canvases[i].BoxRows()
		if diff > 0 {
			fill := fill.NewEmpty()
			fc := fill.Render(gowid.RenderBox{canvases[i].BoxColumns(), diff}, gowid.NotSelected, app)
			canvases[i].AppendBelow(fc, false, false)
		} else if diff < 0 {
			fill := fill.NewEmpty()
			fc := fill.Render(gowid.RenderBox{canvas.BoxColumns(), -diff}, gowid.NotSelected, app)
			canvas.AppendBelow(fc, false, false)
		}
		canvas.AppendRight(canvases[i], false)
	}

	if cols, ok := size.(gowid.IColumns); ok {
		canvas.ExtendRight(gowid.EmptyLine(cols.Columns() - canvas.BoxColumns()))
		if rows, ok2 := size.(gowid.IRenderBox); ok2 && canvas.BoxRows() < rows.BoxRows() {
			gowid.AppendBlankLines(canvas, rows.BoxRows()-canvas.BoxRows())
		}
	}

	gowid.MakeCanvasRightSize(canvas, size)
	return canvas
}

var AllChildrenMaxDimension = fmt.Errorf("All columns widgets were rendered Max, so there is no max height to use.")

// RenderSubWidgets returns an array of canvases for each of the subwidgets, rendering them
// with in the context of a column with the provided size and focus.
func RenderSubWidgets(w IWidget, size gowid.IRenderSize, focus gowid.Selector, rfocus [][]string, app gowid.IApp) []gowid.ICanvas {
	subs := w.SubWidgets()
	wlen := len(subs)
	canvases := make([]gowid.ICanvas, wlen)
	if wlen == 0 {
		return canvases
	}

	curMax := -1
	dims := w.Dimensions()
	weights := w.WidgetWidths(size, focus, app)
	for i := 0; i < wlen; i++ {
		subSize := w.SubWidgetSize(size, weights[i], subs[i], dims[i])
		canvases[i] = subs[i].Render(subSize, focus, wid.WithFocus(app, rfocus[i]))
		if canvases[i].BoxRows() > curMax {
			curMax = canvases[i].BoxRows()
		}
	}
	if curMax == -1 {
		panic(AllChildrenMaxDimension)
	}

	return canvases
}

// RenderedSubWidgetsSizes returns an array of boxes that bound each of the subwidgets as they
// would be rendered with the given size and focus.
func RenderedSubWidgetsSizes(w IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) []gowid.IRenderBox {
	subs := w.SubWidgets()
	dims := w.Dimensions()
	wlen := len(subs)

	curMax := -1
	boxes := make([]gowid.IRenderBox, wlen)
	weights := w.WidgetWidths(size, focus, app)
	for i := 0; i < wlen; i++ {
		subSize := w.SubWidgetSize(size, weights[i], subs[i], dims[i])
		c := subs[i].RenderSize(subSize, focus, app)
		boxes[i] = gowid.RenderBox{weights[i], c.BoxRows()}
		if boxes[i].BoxRows() > curMax {
			curMax = boxes[i].BoxRows()
		}
	}
	if curMax == -1 {
		panic(AllChildrenMaxDimension)
	}

	return boxes
}
