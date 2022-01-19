package rvt

import (
	"github.com/gdamore/tcell/v2"
)

type RemoteEvent struct {
	ID string
	tcell.Event
}

func EventToProto(event tcell.Event) *EventMessage {
	switch evt := event.(type) {
	case *tcell.EventMouse:
		x, y := evt.Position()
		return &EventMessage{Event: &EventMessage_Mouse{
			Mouse: &EventMouse{
				X:          int32(x),
				Y:          int32(y),
				ButtonMask: int32(evt.Buttons()),
				ModMask:    int32(evt.Modifiers()),
			},
		}}
	case *tcell.EventKey:
		return &EventMessage{Event: &EventMessage_Key{
			Key: &EventKey{
				Key:     int32(evt.Key()),
				Rune:    int32(evt.Rune()),
				ModMask: int32(evt.Modifiers()),
			},
		}}
	case *tcell.EventResize:
		// width, height := evt.Size()
		// return &EventMessage{Event: &EventMessage_Resize{
		// 	Resize: &EventResize{
		// 		Width:  int32(width),
		// 		Height: int32(height),
		// 	},
		// }}
	case *tcell.EventPaste:
		return &EventMessage{Event: &EventMessage_Paste{
			Paste: &EventPaste{
				Start: evt.Start(),
			},
		}}
	}
	return nil
}

func ProtoToEvent(msg *EventMessage) tcell.Event {
	switch event := msg.Event.(type) {
	case *EventMessage_Mouse:
		return tcell.NewEventMouse(
			int(event.Mouse.X),
			int(event.Mouse.Y),
			tcell.ButtonMask(event.Mouse.ButtonMask),
			tcell.ModMask(event.Mouse.ModMask),
		)
	case *EventMessage_Key:
		return tcell.NewEventKey(
			tcell.Key(event.Key.Key),
			rune(event.Key.Rune),
			tcell.ModMask(event.Key.ModMask),
		)
	case *EventMessage_Resize:
		return tcell.NewEventResize(
			int(event.Resize.Width),
			int(event.Resize.Height),
		)
	case *EventMessage_Paste:
		return tcell.NewEventPaste(event.Paste.Start)
	}
	return nil
}

func ScreenToRender(s tcell.Screen) *RenderMessage {
	cols, rows := s.Size()

	var glyphs []*Glyph
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			mainc, combc, style, width := s.GetContent(x, y)
			fg, bg, attr := style.Decompose()

			var combi []int32
			for _, c := range combc {
				combi = append(combi, int32(c))
			}

			glyphs = append(glyphs, &Glyph{
				X:        int32(x),
				Y:        int32(y),
				Mainc:    int32(mainc),
				Combc:    combi,
				Fg:       uint64(fg),
				Bg:       uint64(bg),
				AttrMask: int32(attr),
				Width:    int32(width),
			})
		}
	}

	return &RenderMessage{
		Cols:   int32(cols),
		Rows:   int32(rows),
		Glyphs: glyphs,
	}
}

func RenderToScreen(msg *RenderMessage, s tcell.Screen) {
	cols, rows := s.Size()
	for _, glyph := range msg.Glyphs {
		x, y := int(glyph.X), int(glyph.Y)
		if x >= cols || y >= rows {
			continue
		}

		mainc := rune(glyph.Mainc)
		var combc []rune
		for _, i := range glyph.Combc {
			combc = append(combc, rune(i))
		}

		style := tcell.Style{}.
			Foreground(tcell.Color(glyph.Fg)).
			Background(tcell.Color(glyph.Bg)).
			Attributes(tcell.AttrMask(glyph.AttrMask))

		s.SetContent(x, y, mainc, combc, style)
	}
	s.Show()
}
