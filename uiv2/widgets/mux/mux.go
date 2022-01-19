package mux

import (
	"fmt"
	"os"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/uiv2/widgets/terminal"
)

var log *os.File

func init() {
	log, _ = os.Create("console.log")
}

var (
	Frame = framed.FrameRunes{'┌', '┐', '└', '┘', '─', '─', '│', '│'}
)

type IWidget interface {
	gowid.ICompositeWidget
	IMux
}

type IMux interface {
	SelectedPane() IPane
	VerticalSplit(IPane, gowid.IApp)
	HorizontalSplit(IPane, gowid.IApp)
	KillPane(IPane, gowid.IApp)
}

type IPane interface {
	gowid.IWidget
}

type pane struct {
	gowid.IContainerWidget
	count int
}

var count = 0

func NewPane() IPane {
	var (
		term gowid.IWidget
		err  error
	)
	term, err = terminal.New()
	if err != nil {
		term = text.New(err.Error())
	}
	count++
	return &pane{
		IContainerWidget: &gowid.ContainerWidget{
			IWidget: styled.NewFocus(
				framed.New(term, framed.Options{
					Frame:       Frame,
					TitleWidget: text.New(fmt.Sprintf(" %d ", count)),
				}),
				gowid.MakeForeground(gowid.ColorBlue),
			),
			D: gowid.RenderWithWeight{1},
		},
		count: count,
	}
}

func (p *pane) String() string {
	return fmt.Sprintf("pane[%d]", p.count)
}

type Widget struct {
	gowid.IWidget
	selectedPane IPane
}

var _ gowid.IWidget = (*Widget)(nil)

func New() *Widget {
	pane := NewPane()
	return &Widget{
		IWidget:      pane,
		selectedPane: pane,
	}
}

func (w *Widget) String() string {
	return fmt.Sprintf("%s", w.IWidget)
}

func (w *Widget) SelectedPane() IPane {
	return w.selectedPane
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

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	return UserInput(w, ev, size, focus, app)
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

func (w *Widget) FocusPaneFunction(app gowid.IApp, focusParent gowid.IWidget) {
	cw, ok := focusParent.(gowid.ICompositeMultipleFocus)
	if ok {
		focused := cw.SubWidgets()[cw.Focus()]
		log.Write([]byte(fmt.Sprintf("focus pane function | parent %s | focus %d | child %s\n", focusParent, cw.Focus(), focused)))
		p, ok := focused.(*pane)
		if ok {
			log.Write([]byte(fmt.Sprintf("[FocusPaneFunction] parent %s focus changed to %d\n", focusParent, cw.Focus())))
			w.selectedPane = p
		} else {
			log.Write([]byte(fmt.Sprintf("focus parent focus subwidget is not pane %s\n", focused)))
		}
	} else {
		log.Write([]byte(fmt.Sprintf("focus parent is not composite focus %s\n", focusParent)))
	}
}

func (w *Widget) VerticalSplit(p IPane, app gowid.IApp) {
	log.Write([]byte(fmt.Sprintf("vertical split %s\n", p)))
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	widgets := []gowid.IWidget{p, NewPane()}
	containers := make([]gowid.IContainerWidget, len(widgets))
	for i, widget := range widgets {
		containers[i] = widget.(gowid.IContainerWidget)
	}
	if _, ok := parent.(*pane); ok || parent == nil {
		hlist := columns.New(containers)
		hlist.OnFocusChanged(gowid.WidgetCallback{"cb", w.FocusPaneFunction})
		hlist.SetFocus(app, 1)
		w.SetSubWidget(&gowid.ContainerWidget{
			       IWidget: hlist,
			       D: gowid.RenderWithWeight{1},
		}, app)
		return
	}
	if _, ok := parent.(*columns.Widget); !ok {
		hlist := columns.New(containers)
		hlist.OnFocusChanged(gowid.WidgetCallback{"cb", w.FocusPaneFunction})
		hlist.SetFocus(app, 1)
		widgets = []gowid.IWidget{
			&gowid.ContainerWidget{
				IWidget: hlist,
				D:       gowid.RenderWithWeight{1},
			},
		}
	}

	w.split(parent, p, app, widgets)
}

func (w *Widget) HorizontalSplit(p IPane, app gowid.IApp) {
	log.Write([]byte(fmt.Sprintf("horizontal split %s\n", p)))
	parent := FindParentInHierarchy(w.IWidget, MatchWidget(p))

	widgets := []gowid.IWidget{p, NewPane()}
	containers := make([]gowid.IContainerWidget, len(widgets))
	for i, widget := range widgets {
		containers[i] = widget.(gowid.IContainerWidget)
	}
	if _, ok := parent.(*pane); ok || parent == nil {
		vlist := pile.New(containers)
		vlist.OnFocusChanged(gowid.WidgetCallback{"cb", w.FocusPaneFunction})
		vlist.SetFocus(app, 1)
		w.SetSubWidget(&gowid.ContainerWidget{
			       IWidget: vlist,
			       D: gowid.RenderWithWeight{1},
		}, app)
		return
	}
	if _, ok := parent.(*pile.Widget); !ok {
		vlist := pile.New(containers)
		vlist.OnFocusChanged(gowid.WidgetCallback{"cb", w.FocusPaneFunction})
		vlist.SetFocus(app, 1)
		widgets = []gowid.IWidget{
			&gowid.ContainerWidget{
				IWidget: vlist,
				D:       gowid.RenderWithWeight{1},
			},
		}
	}

	w.split(parent, p, app, widgets)
}

func (w *Widget) KillPane(p IPane, app gowid.IApp) {
	log.Write([]byte(fmt.Sprintf("kill pane %s\n", p)))
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
		parent.(gowid.IFocus).SetFocus(app, focus)
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
		w.selectedPane = sibling
		w.SetSubWidget(sibling, app)
		return
	}
	i, _ = FindNextWidgetFrom(grandparent.(gowid.ICompositeMultiple), func(w gowid.IWidget) bool {
		if cw, ok := w.(gowid.IComposite); ok {
			w = cw.SubWidget()
		}
		return w == parent
	})
	insertSubwidgets(grandparent, app, i, []gowid.IWidget{sibling})
}

func (w *Widget) split(parent gowid.IWidget, p IPane, app gowid.IApp, widgets []gowid.IWidget) {
	i, _ := FindNextWidgetFrom(parent.(gowid.ICompositeMultiple), func(w gowid.IWidget) bool {
		return w == p
	})
	insertSubwidgets(parent, app, i, widgets)
}

func insertSubwidgets(w gowid.IWidget, app gowid.IApp, i int, widgets []gowid.IWidget) {
	children := w.(gowid.ICompositeMultiple).SubWidgets()
	focus := i + len(widgets) - 1
	if i+1 < len(children) {
		widgets = append(widgets, children[i+1:]...)
	}
	w.(gowid.ISettableSubWidgets).SetSubWidgets(append(children[:i], widgets...), app)
	w.(gowid.IFocus).SetFocus(app, focus)
	log.Write([]byte(fmt.Sprintf("insert into parent %s (%d widgets) focus to %d\n", w, len(widgets), focus)))
}

func UserInput(w IWidget, ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	handled := gowid.UserInputIfSelectable(w.SubWidget(), ev, size, focus, app)
	if handled {
		return true
	}
	if evk, ok := ev.(*tcell.EventKey); ok {
		switch evk.Key() {
		case tcell.KeyRune:
			log.Write([]byte(fmt.Sprintf("before [%c]: %s\n", evk.Rune(), w)))
			handled = true
			switch evk.Rune() {
			case '%':
				w.VerticalSplit(w.SelectedPane(), app)
			case '"':
				w.HorizontalSplit(w.SelectedPane(), app)
			case 'x':
				w.KillPane(w.SelectedPane(), app)
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
