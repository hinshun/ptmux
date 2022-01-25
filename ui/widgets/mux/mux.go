package mux

import (
	"fmt"

	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
	"github.com/hinshun/ptmux/ui/widgets/columns"
	"github.com/hinshun/ptmux/ui/widgets/pane"
	"github.com/hinshun/ptmux/ui/widgets/pile"
)

type IWidget interface {
	gowid.ICompositeWidget
	IMux
}

type IMux interface {
	FocusedPane(id string) *pane.Widget
	VerticalSplit(id string, p *pane.Widget, app gowid.IApp)
	HorizontalSplit(id string, p *pane.Widget, app gowid.IApp)
	KillPane(id string, p *pane.Widget, app gowid.IApp)
}

func (w *Widget) NewPane(id string) *pane.Widget {
	p := pane.New(w.defaultID, id)

	term := p.GetTerminal()
	if term != nil {
		term.OnProcessExited(gowid.WidgetCallbackExt{"cb",
			func(app gowid.IApp, _ gowid.IWidget, data ...interface{}) {
				lastID := data[0].(string)
				w.KillPane(lastID, p, app)
			},
		})
	}

	return p
}

type Widget struct {
	gowid.IWidget
	defaultID string
}

var _ gowid.IWidget = (*Widget)(nil)

func New(defaultID string) *Widget {
	w := &Widget{defaultID: defaultID}
	w.IWidget = w.NewPane(defaultID)
	return w
}

func (w *Widget) String() string {
	return fmt.Sprintf("%s", w.IWidget)
}

func (w *Widget) FocusedPane(id string) *pane.Widget {
	return findFocusedPane(id, w.IWidget)
}

func findFocusedPane(id string, w gowid.IWidget) *pane.Widget {
	if p, ok := w.(*pane.Widget); ok {
		return p
	}

	if cw, ok := w.(gowid.IComposite); ok {
		w = cw.SubWidget()
	}

	if cmf, ok := w.(wid.ICompositeMultipleFocus); ok {
		focus := cmf.SubWidgets()[cmf.Focus(id)]
		return findFocusedPane(id, focus)
	}

	return nil
}

func (w *Widget) SubWidget() gowid.IWidget {
	return w.IWidget
}

func (w *Widget) SetSubWidget(wi gowid.IWidget, app gowid.IApp) {
	w.IWidget = wi
}

func (w *Widget) SubWidgetSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderSize {
	return w.SubWidget().RenderSize(size, focus, app)
}

func (w *Widget) VerticalSplit(id string, p *pane.Widget, app gowid.IApp) {
	if p == nil {
		return
	}
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	widgets := []gowid.IWidget{p, w.NewPane(id)}
	containers := make([]gowid.IContainerWidget, len(widgets))
	for i, widget := range widgets {
		containers[i] = widget.(gowid.IContainerWidget)
	}

	if parent == nil {
		hlist := columns.New(w.defaultID, containers)
		ids := app.(wid.IP2PApp).IDs()
		for _, id := range ids {
			hlist.SetFocus(id, 0)
		}
		hlist.SetFocus(id, 1)
		w.SetSubWidget(&gowid.ContainerWidget{
			IWidget: hlist,
			D:       gowid.RenderWithWeight{1},
		}, app)
		return
	}
	// If parent is not a column.
	if _, ok := parent.(*columns.Widget); !ok {
		hlist := columns.New(w.defaultID, containers)
		hlist.SetFocus(id, 1)
		widgets = []gowid.IWidget{
			&gowid.ContainerWidget{
				IWidget: hlist,
				D:       gowid.RenderWithWeight{1},
			},
		}
	}

	w.split(id, parent, p, app, widgets)
}

func (w *Widget) HorizontalSplit(id string, p *pane.Widget, app gowid.IApp) {
	if p == nil {
		return
	}
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	widgets := []gowid.IWidget{p, w.NewPane(id)}
	containers := make([]gowid.IContainerWidget, len(widgets))
	for i, widget := range widgets {
		containers[i] = widget.(gowid.IContainerWidget)
	}

	ids := app.(wid.IP2PApp).IDs()
	if parent == nil {
		vlist := pile.New(w.defaultID, containers)
		for _, id := range ids {
			vlist.SetFocus(id, 0)
		}
		vlist.SetFocus(id, 1)
		w.SetSubWidget(&gowid.ContainerWidget{
			IWidget: vlist,
			D:       gowid.RenderWithWeight{1},
		}, app)
		return
	}
	// If parent is not a pile.
	if _, ok := parent.(*pile.Widget); !ok {
		vlist := pile.New(w.defaultID, containers)
		for _, id := range ids {
			vlist.SetFocus(id, 0)
		}
		vlist.SetFocus(id, 1)
		widgets = []gowid.IWidget{
			&gowid.ContainerWidget{
				IWidget: vlist,
				D:       gowid.RenderWithWeight{1},
			},
		}
	}

	w.split(id, parent, p, app, widgets)
}

func (w *Widget) KillPane(id string, p *pane.Widget, app gowid.IApp) {
	if p == nil {
		return
	}
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	// If there is only one pane, then parent will be nil.
	if parent == nil {
		app.Quit()
		return
	}
	i, _ := FindNextWidgetFrom(parent.(gowid.ICompositeMultiple), func(w gowid.IWidget) bool {
		return w == p
	})

	children := parent.(gowid.ICompositeMultiple).SubWidgets()

	// If killing pane leaves more than one child behind, just remove from its parent.
	if len(children) > 2 {
		// Find the sibling pane that should be the new selected pane.
		focus := 0
		if i > 0 {
			focus = i - 1
		}

		f := parent.(wid.IFocus)
		rfocus := f.ReverseFocus()

		// Build the list of ids for each widget that will be shifted by one after
		// the delete.
		var shifts [][]string
		for j := focus + 1; j < len(children); j++ {
			shifts = append(shifts, rfocus[j])
		}

		parent.(gowid.ISettableSubWidgets).SetSubWidgets(append(children[:i], children[i+1:]...), app)

		// Shift all ids affected by the delete.
		for j, ids := range shifts {
			for _, id := range ids {
				f.SetFocus(id, j+focus)
			}
		}
		return
	}

	sibling := children[1]
	if i > 0 {
		sibling = children[0]
	}

	// Otherwise, there is only one child left. The child should replace its parent.
	grandparent := FindParentInHierarchy(w.IWidget, MatchWidget(parent))
	if grandparent == nil {
		w.SetSubWidget(sibling, app)
		return
	}

	i, _ = FindNextWidgetFrom(grandparent.(gowid.ICompositeMultiple), func(w gowid.IWidget) bool {
		if cw, ok := w.(gowid.IComposite); ok {
			w = cw.SubWidget()
		}
		return w == parent
	})

	f := grandparent.(wid.IFocus)
	oldFocus := f.Focus(id)
	ids := f.ReverseFocus()[oldFocus]
	focus := insertSubwidgets(id, grandparent, app, i, []gowid.IWidget{sibling})
	for _, id := range ids {
		f.SetFocus(id, focus)
	}
}

func (w *Widget) split(id string, parent gowid.IWidget, p *pane.Widget, app gowid.IApp, widgets []gowid.IWidget) {
	i, _ := FindNextWidgetFrom(parent.(gowid.ICompositeMultiple), func(w gowid.IWidget) bool {
		return w == p
	})

	f := parent.(wid.IFocus)
	oldFocus := f.Focus(id)
	rfocus := f.ReverseFocus()

	// Build the list of ids for each widget that will be shifted by one after
	// the insert.
	var shifts [][]string
	for j := oldFocus + 1; j < len(parent.(gowid.ICompositeMultiple).SubWidgets()); j++ {
		shifts = append(shifts, rfocus[j])
	}

	// Insert the new subwidgets.
	focus := insertSubwidgets(id, parent, app, i, widgets)

	// First shift all ids affected by the insert.
	for j, ids := range shifts {
		for _, id := range ids {
			f.SetFocus(id, focus+j+1)
		}
	}

	// Focus the original id that executed the split.
	f.SetFocus(id, focus)
}

func insertSubwidgets(id string, w gowid.IWidget, app gowid.IApp, i int, widgets []gowid.IWidget) (focus int) {
	children := w.(gowid.ICompositeMultiple).SubWidgets()
	focus = i + len(widgets) - 1
	if i+1 < len(children) {
		widgets = append(widgets, children[i+1:]...)
	}
	w.(gowid.ISettableSubWidgets).SetSubWidgets(append(children[:i], widgets...), app)
	return focus
}

func MatchWidget(m gowid.IWidget) WidgetsPredicate {
	if cm, ok := m.(gowid.IComposite); ok {
		m = cm.SubWidget()
	}
	return func(widgets []gowid.IWidget) bool {
		for _, w := range widgets {
			if cw, ok := w.(gowid.IComposite); ok {
				w = cw.SubWidget()
			}
			if w == m {
				return true
			}
		}
		return false
	}
}

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	handled := gowid.UserInputIfSelectable(w.SubWidget(), ev, size, focus, app)
	if handled {
		return true
	}

	evt := ev
	id := w.defaultID
	if evr, ok := ev.(*rvt.RemoteEvent); ok {
		evt = evr.Event
		id = evr.ID
	}

	if evk, ok := evt.(*tcell.EventKey); ok {
		switch evk.Key() {
		case tcell.KeyRune:
			handled = true
			switch evk.Rune() {
			case '%':
				w.VerticalSplit(id, w.FocusedPane(id), app)
			case '"':
				w.HorizontalSplit(id, w.FocusedPane(id), app)
			case 'x':
				w.KillPane(id, w.FocusedPane(id), app)
			default:
				handled = false
			}
		}
	}
	return handled
}

type WidgetsPredicate func([]gowid.IWidget) bool

func FindParentInHierarchy(w gowid.IWidget, pred WidgetsPredicate) gowid.IWidget {
	var res, parent gowid.IWidget
	next := []gowid.IWidget{w}
	for len(next) > 0 {
		w := next[0]
		next = next[1:]
		if cw, ok := w.(gowid.IComposite); ok {
			w = cw.SubWidget()
		}
		if cw, ok := w.(gowid.ICompositeMultiple); ok {
			parent = w
			widgets := cw.SubWidgets()
			if pred(widgets) {
				res = parent
				break
			}
			next = append(next, widgets...)
		}
	}
	return res
}

func FindNextWidgetFrom(w gowid.ICompositeMultiple, pred gowid.WidgetPredicate) (int, bool) {
	dup := gowid.CopyWidgets(w.SubWidgets())
	return FindWidgetIndex(dup, pred)
}

func FindWidgetIndex(w []gowid.IWidget, pred gowid.WidgetPredicate) (int, bool) {
	if len(w) == 0 {
		return -1, false
	}
	pos := -1
	for {
		pos++
		if pos == len(w) {
			return -1, false
		}
		if w[pos] != nil && pred(w[pos]) {
			return pos, true
		}
	}
}
