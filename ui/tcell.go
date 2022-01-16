package ui

import (
	"fmt"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/terminfo"
	"github.com/hinshun/vt10x"
)

type EventNotSupported struct {
	Event tcell.Event
}

var _ error = EventNotSupported{}

func (e EventNotSupported) Error() string {
	return fmt.Sprintf("Terminal input %v not supported yet", e.Event)
}

type MouseState struct {
	MouseLeftClicked   bool
	MouseMiddleClicked bool
	MouseRightClicked  bool
}

func (m MouseState) NoButtonClicked() bool {
	return !m.LeftIsClicked() && !m.MiddleIsClicked() && !m.RightIsClicked()
}

func (m MouseState) LeftIsClicked() bool {
	return m.MouseLeftClicked
}

func (m MouseState) MiddleIsClicked() bool {
	return m.MouseMiddleClicked
}

func (m MouseState) RightIsClicked() bool {
	return m.MouseRightClicked
}

// eventToBytes converts TCell's representation of a terminal event to
// the string of bytes that would be the equivalent event according to the
// supplied Terminfo object. It returns a tuple of the byte slice
// representing the terminal event (if successful), and a bool (denoting
// success or failure). This function is used by the TerminalWidget. Its
// subprocess is connected to a tty controlled by gowid. Events from the
// user are parsed by gowid via TCell - they are then translated by this
// function before being written to the TerminalWidget subprocess's tty.
func eventToBytes(view vt10x.View, event tcell.Event, last MouseState, ti *terminfo.Terminfo) (result []byte, success bool) {
	result = make([]byte, 0)
	switch evt := event.(type) {
	case *tcell.EventKey:
		if evt.Key() < ' ' {
			str := []rune{rune(evt.Key())}
			result = append(result, string(str)...)
			success = true
		} else {
			success = true
			switch evt.Key() {
			case tcell.KeyRune:
				str := []rune{evt.Rune()}
				result = append(result, string(str)...)
			case tcell.KeyCR:
				str := []rune{rune(tcell.KeyCR)}
				result = append(result, string(str)...)
			case tcell.KeyF1:
				result = append(result, ti.KeyF1...)
			case tcell.KeyF2:
				result = append(result, ti.KeyF2...)
			case tcell.KeyF3:
				result = append(result, ti.KeyF3...)
			case tcell.KeyF4:
				result = append(result, ti.KeyF4...)
			case tcell.KeyF5:
				result = append(result, ti.KeyF5...)
			case tcell.KeyF6:
				result = append(result, ti.KeyF6...)
			case tcell.KeyF7:
				result = append(result, ti.KeyF7...)
			case tcell.KeyF8:
				result = append(result, ti.KeyF8...)
			case tcell.KeyF9:
				result = append(result, ti.KeyF9...)
			case tcell.KeyF10:
				result = append(result, ti.KeyF10...)
			case tcell.KeyF11:
				result = append(result, ti.KeyF11...)
			case tcell.KeyF12:
				result = append(result, ti.KeyF12...)
			case tcell.KeyF13:
				result = append(result, ti.KeyF13...)
			case tcell.KeyF14:
				result = append(result, ti.KeyF14...)
			case tcell.KeyF15:
				result = append(result, ti.KeyF15...)
			case tcell.KeyF16:
				result = append(result, ti.KeyF16...)
			case tcell.KeyF17:
				result = append(result, ti.KeyF17...)
			case tcell.KeyF18:
				result = append(result, ti.KeyF18...)
			case tcell.KeyF19:
				result = append(result, ti.KeyF19...)
			case tcell.KeyF20:
				result = append(result, ti.KeyF20...)
			case tcell.KeyF21:
				result = append(result, ti.KeyF21...)
			case tcell.KeyF22:
				result = append(result, ti.KeyF22...)
			case tcell.KeyF23:
				result = append(result, ti.KeyF23...)
			case tcell.KeyF24:
				result = append(result, ti.KeyF24...)
			case tcell.KeyF25:
				result = append(result, ti.KeyF25...)
			case tcell.KeyF26:
				result = append(result, ti.KeyF26...)
			case tcell.KeyF27:
				result = append(result, ti.KeyF27...)
			case tcell.KeyF28:
				result = append(result, ti.KeyF28...)
			case tcell.KeyF29:
				result = append(result, ti.KeyF29...)
			case tcell.KeyF30:
				result = append(result, ti.KeyF30...)
			case tcell.KeyF31:
				result = append(result, ti.KeyF31...)
			case tcell.KeyF32:
				result = append(result, ti.KeyF32...)
			case tcell.KeyF33:
				result = append(result, ti.KeyF33...)
			case tcell.KeyF34:
				result = append(result, ti.KeyF34...)
			case tcell.KeyF35:
				result = append(result, ti.KeyF35...)
			case tcell.KeyF36:
				result = append(result, ti.KeyF36...)
			case tcell.KeyF37:
				result = append(result, ti.KeyF37...)
			case tcell.KeyF38:
				result = append(result, ti.KeyF38...)
			case tcell.KeyF39:
				result = append(result, ti.KeyF39...)
			case tcell.KeyF40:
				result = append(result, ti.KeyF40...)
			case tcell.KeyF41:
				result = append(result, ti.KeyF41...)
			case tcell.KeyF42:
				result = append(result, ti.KeyF42...)
			case tcell.KeyF43:
				result = append(result, ti.KeyF43...)
			case tcell.KeyF44:
				result = append(result, ti.KeyF44...)
			case tcell.KeyF45:
				result = append(result, ti.KeyF45...)
			case tcell.KeyF46:
				result = append(result, ti.KeyF46...)
			case tcell.KeyF47:
				result = append(result, ti.KeyF47...)
			case tcell.KeyF48:
				result = append(result, ti.KeyF48...)
			case tcell.KeyF49:
				result = append(result, ti.KeyF49...)
			case tcell.KeyF50:
				result = append(result, ti.KeyF50...)
			case tcell.KeyF51:
				result = append(result, ti.KeyF51...)
			case tcell.KeyF52:
				result = append(result, ti.KeyF52...)
			case tcell.KeyF53:
				result = append(result, ti.KeyF53...)
			case tcell.KeyF54:
				result = append(result, ti.KeyF54...)
			case tcell.KeyF55:
				result = append(result, ti.KeyF55...)
			case tcell.KeyF56:
				result = append(result, ti.KeyF56...)
			case tcell.KeyF57:
				result = append(result, ti.KeyF57...)
			case tcell.KeyF58:
				result = append(result, ti.KeyF58...)
			case tcell.KeyF59:
				result = append(result, ti.KeyF59...)
			case tcell.KeyF60:
				result = append(result, ti.KeyF60...)
			case tcell.KeyF61:
				result = append(result, ti.KeyF61...)
			case tcell.KeyF62:
				result = append(result, ti.KeyF62...)
			case tcell.KeyF63:
				result = append(result, ti.KeyF63...)
			case tcell.KeyF64:
				result = append(result, ti.KeyF64...)
			case tcell.KeyInsert:
				result = append(result, ti.KeyInsert...)
			case tcell.KeyDelete:
				result = append(result, ti.KeyDelete...)
			case tcell.KeyHome:
				result = append(result, ti.KeyHome...)
			case tcell.KeyEnd:
				result = append(result, ti.KeyEnd...)
			case tcell.KeyHelp:
				result = append(result, ti.KeyHelp...)
			case tcell.KeyPgUp:
				result = append(result, ti.KeyPgUp...)
			case tcell.KeyPgDn:
				result = append(result, ti.KeyPgDn...)
			case tcell.KeyUp:
				result = append(result, ti.KeyUp...)
			case tcell.KeyDown:
				result = append(result, ti.KeyDown...)
			case tcell.KeyLeft:
				result = append(result, ti.KeyLeft...)
			case tcell.KeyRight:
				result = append(result, ti.KeyRight...)
			case tcell.KeyBacktab:
				result = append(result, ti.KeyBacktab...)
			case tcell.KeyExit:
				result = append(result, ti.KeyExit...)
			case tcell.KeyClear:
				result = append(result, ti.KeyClear...)
			case tcell.KeyPrint:
				result = append(result, ti.KeyPrint...)
			case tcell.KeyCancel:
				result = append(result, ti.KeyCancel...)
			case tcell.KeyDEL:
				result = append(result, ti.KeyBackspace...)
			case tcell.KeyBackspace:
				result = append(result, ti.KeyBackspace...)
			default:
				success = false
				panic(EventNotSupported{Event: evt})
			}
		}
	case *tcell.EventMouse:
		var data string

		btnind := 0
		switch evt.Buttons() {
		case tcell.Button1:
			btnind = 0
		case tcell.Button2:
			btnind = 1
		case tcell.Button3:
			btnind = 2
		case tcell.WheelUp:
			btnind = 64
		case tcell.WheelDown:
			btnind = 65
		}

		lastind := 0
		if last.LeftIsClicked() {
			lastind = 0
		} else if last.MiddleIsClicked() {
			lastind = 1
		} else if last.RightIsClicked() {
			lastind = 2
		}

		switch evt.Buttons() {
		case tcell.Button1, tcell.Button2, tcell.Button3, tcell.WheelUp, tcell.WheelDown:
			mx, my := evt.Position()
			btn := btnind
			if (last.LeftIsClicked() && (evt.Buttons() == tcell.Button1)) ||
				(last.MiddleIsClicked() && (evt.Buttons() == tcell.Button2)) ||
				(last.RightIsClicked() && (evt.Buttons() == tcell.Button3)) {
				// Assume mouse pointer moved with button down i.e. drag
				btn += 32
			}
			if view.Mode()&vt10x.ModeMouseSgr == 0 {
				data = fmt.Sprintf("\033[<%d;%d;%dM", btn, mx+1, my+1)
			} else {
				data = fmt.Sprintf("\033[M%c%c%c", btn+32, mx+33, my+33)
			}
			result = append(result, data...)
			success = true
		case tcell.ButtonNone:
			// TODO - how to report no press?
			mx, my := evt.Position()

			if last.LeftIsClicked() || last.MiddleIsClicked() || last.RightIsClicked() {
				// 0 means left mouse button, m means released
				if view.Mode()&vt10x.ModeMouseSgr == 0 {
					data = fmt.Sprintf("\033[<%d;%d;%dm", lastind, mx+1, my+1)
				} else if view.Mode()&vt10x.ModeMouseMany == 0 {
					data = fmt.Sprintf("\033[M%c%c%c", 35, mx+33, my+33)
				}
			} else if view.Mode()&vt10x.ModeMouseMany == 0 {
				if view.Mode()&vt10x.ModeMouseSgr == 0 {
					// +32 for motion, +3 for no button
					data = fmt.Sprintf("\033[<35;%d;%dm", mx+1, my+1)
				} else {
					data = fmt.Sprintf("\033[M%c%c%c", 35+32, mx+33, my+33)
				}
			}
			result = append(result, data...)
			success = true
		}
	}
	return result, success
}
