package rvt

import "github.com/gdamore/tcell/v2"

func tcellEventToProto(event tcell.Event) *TcellMessage {
	switch evt := event.(type) {
	case *tcell.EventMouse:
		x, y := evt.Position()
		return &TcellMessage{TcellEvent: &TcellMessage_Mouse{
			Mouse: &EventMouse{
				X:          int64(x),
				Y:          int64(y),
				ButtonMask: int32(evt.Buttons()),
				ModMask:    int32(evt.Modifiers()),
			},
		}}
	case *tcell.EventKey:
		return &TcellMessage{TcellEvent: &TcellMessage_Key{
			Key: &EventKey{
				Key:     int32(evt.Key()),
				Rune:    int32(evt.Rune()),
				ModMask: int32(evt.Modifiers()),
			},
		}}
	case *tcell.EventResize:
		width, height := evt.Size()
		return &TcellMessage{TcellEvent: &TcellMessage_Resize{
			Resize: &EventResize{
				Width:  int64(width),
				Height: int64(height),
			},
		}}
	case *tcell.EventPaste:
		return &TcellMessage{TcellEvent: &TcellMessage_Paste{
			Paste: &EventPaste{
				Start: evt.Start(),
			},
		}}
	}
	return nil
}

func protoEventToTcell(msg *TcellMessage) tcell.Event {
	switch event := msg.TcellEvent.(type) {
	case *TcellMessage_Mouse:
		return tcell.NewEventMouse(
			int(event.Mouse.X),
			int(event.Mouse.Y),
			tcell.ButtonMask(event.Mouse.ButtonMask),
			tcell.ModMask(event.Mouse.ModMask),
		)
	case *TcellMessage_Key:
		return tcell.NewEventKey(
			tcell.Key(event.Key.Key),
			rune(event.Key.Rune),
			tcell.ModMask(event.Key.ModMask),
		)
	case *TcellMessage_Resize:
		return tcell.NewEventResize(
			int(event.Resize.Width),
			int(event.Resize.Height),
		)
	case *TcellMessage_Paste:
		return tcell.NewEventPaste(event.Paste.Start)
	}
	return nil
}
