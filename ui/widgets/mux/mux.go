package mux

import (
	"fmt"
	"os"

	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
	"github.com/hinshun/ptmux/ui/widgets/columns"
	"github.com/hinshun/ptmux/ui/widgets/pane"
	"github.com/hinshun/ptmux/ui/widgets/pile"
)

var log *os.File

func init() {
	log, _ = os.Create("mux.log")
}

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

func (w *Widget) NewPane() *pane.Widget {
	p := pane.New(w.defaultID)

	term := p.GetTerminal()
	if term != nil {
		term.OnProcessExited(gowid.WidgetCallback{"cb",
			func(app gowid.IApp, _ gowid.IWidget) {
				w.KillPane("self", p, app)
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
	w.IWidget = w.NewPane()
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
	log.Write([]byte(fmt.Sprintf("vertical split %s\n", p)))
	if p == nil {
		return
	}
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	widgets := []gowid.IWidget{p, w.NewPane()}
	containers := make([]gowid.IContainerWidget, len(widgets))
	for i, widget := range widgets {
		containers[i] = widget.(gowid.IContainerWidget)
	}
	if _, ok := parent.(*pane.Widget); ok || parent == nil {
		hlist := columns.New(w.defaultID, containers)
		hlist.SetFocus(id, 1)
		w.SetSubWidget(&gowid.ContainerWidget{
			IWidget: hlist,
			D:       gowid.RenderWithWeight{1},
		}, app)
		return
	}
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
	log.Write([]byte(fmt.Sprintf("horizontal split %s\n", p)))
	if p == nil {
		return
	}
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	widgets := []gowid.IWidget{p, w.NewPane()}
	containers := make([]gowid.IContainerWidget, len(widgets))
	for i, widget := range widgets {
		containers[i] = widget.(gowid.IContainerWidget)
	}
	if _, ok := parent.(*pane.Widget); ok || parent == nil {
		vlist := pile.New(w.defaultID, containers)
		vlist.SetFocus(id, 1)
		w.SetSubWidget(&gowid.ContainerWidget{
			IWidget: vlist,
			D:       gowid.RenderWithWeight{1},
		}, app)
		return
	}
	if _, ok := parent.(*pile.Widget); !ok {
		vlist := pile.New(w.defaultID, containers)
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
	log.Write([]byte(fmt.Sprintf("kill pane %s\n", p)))
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

		parent.(gowid.ISettableSubWidgets).SetSubWidgets(append(children[:i], children[i+1:]...), app)
		parent.(wid.IFocus).SetFocus(id, focus)
		log.Write([]byte(fmt.Sprintf("parent %s (more than 2 child) focus to %d\n", parent, focus)))
		return
	}

	sibling := children[1]
	if i > 0 {
		sibling = children[0]
	}
	log.Write([]byte(fmt.Sprintf("sibling %s\n", sibling)))

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
	insertSubwidgets(id, grandparent, app, i, []gowid.IWidget{sibling})
}

func (w *Widget) split(id string, parent gowid.IWidget, p *pane.Widget, app gowid.IApp, widgets []gowid.IWidget) {
	i, _ := FindNextWidgetFrom(parent.(gowid.ICompositeMultiple), func(w gowid.IWidget) bool {
		return w == p
	})
	insertSubwidgets(id, parent, app, i, widgets)
}

func insertSubwidgets(id string, w gowid.IWidget, app gowid.IApp, i int, widgets []gowid.IWidget) {
	children := w.(gowid.ICompositeMultiple).SubWidgets()
	focus := i + len(widgets) - 1
	if i+1 < len(children) {
		widgets = append(widgets, children[i+1:]...)
	}
	w.(gowid.ISettableSubWidgets).SetSubWidgets(append(children[:i], widgets...), app)
	w.(wid.IFocus).SetFocus(id, focus)
	log.Write([]byte(fmt.Sprintf("insert into parent %s (%d widgets) focus to %d\n", w, len(widgets), focus)))
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
			log.Write([]byte(fmt.Sprintf("before [%c]: %s\n", evk.Rune(), w)))
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
			log.Write([]byte(fmt.Sprintf("after [%c]: %s\n\n", evk.Rune(), w)))
		}
	}
	return handled
}

type WidgetsPredicate func([]gowid.IWidget) bool

func FindParentInHierarchy(w gowid.IWidget, pred WidgetsPredicate) gowid.IWidget {
	log.Write([]byte("find parent in hierarchy\n"))
	var res, parent gowid.IWidget
	next := []gowid.IWidget{w}
	for len(next) > 0 {
		w := next[0]
		log.Write([]byte(fmt.Sprintf("next parent %s\n", w)))
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
	log.Write([]byte(fmt.Sprintf("return %s\n", res)))
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
