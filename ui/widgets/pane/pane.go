package pane

import (
	"fmt"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/text"
	"github.com/hinshun/ptmux/ui/widgets/styled"
	"github.com/hinshun/ptmux/ui/widgets/terminal"
)

var (
	Frame = framed.FrameRunes{'┌', '┐', '└', '┘', '─', '─', '│', '│'}
)

type IWidget interface {
	gowid.IWidget
}

type Widget struct {
	*gowid.ContainerWidget
	term *terminal.Widget
}

func New(id string) *Widget {
	var view gowid.IWidget
	term, err := terminal.New(id)
	if err != nil {
		view = text.New(err.Error())
	} else {
		view = term
	}

	frame := framed.New(view, framed.Options{
		Frame: Frame,
		Title: "~",
	})

	w := &Widget{
		ContainerWidget: &gowid.ContainerWidget{
			IWidget: styled.New(id, frame),
			D: gowid.RenderWithWeight{1},
		},
		term: term,
	}

	if term != nil {
		term.OnTitleChanged(gowid.WidgetCallbackExt{"cb",
			func(app gowid.IApp, w gowid.IWidget, data ...interface{}) {
				title, ok := data[0].(string)
				if ok {
					frame.SetTitle(title, app)
				}
			},
		})
	}

	return w
}

func (w *Widget) String() string {
	return fmt.Sprintf("pane[%s]", w.ContainerWidget)
}

func (w *Widget) GetTerminal() *terminal.Widget {
	return w.term
}
