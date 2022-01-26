package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	gowidterminal "github.com/gcla/gowid/widgets/terminal"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/terminfo"
	"github.com/hinshun/ptmux/pkg/vt"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui/wid"
	"github.com/hinshun/vt10x"
)

type TitleChanged struct{}
type ProcessExited struct{}

type IWidget interface {
	io.Writer
	gowid.IWidget
	gowidterminal.IHotKeyProvider
	gowidterminal.IHotKeyPersistence
	MouseSupport() gowidterminal.IMouseSupport
	Terminfo() *terminfo.Terminfo
	HotKeyActive() bool
	SetHotKeyActive(app gowid.IApp, down bool)
	HotKeyDownTime() time.Time
	Scroll(dir ScrollDir, page bool, lines int)
	ResetScroll()
	Scrolling() bool
}

type Widget struct {
	gowidterminal.IHotKeyProvider
	gowidterminal.IHotKeyPersistence
	Callbacks         *gowid.Callbacks
	terminfo          *terminfo.Terminfo
	vt                *vt.VT
	canvas            *Canvas
	width, height     int
	title             string
	defaultID, lastID string
	hotKeyDown        bool
	hotKeyDownTime    time.Time
	hotKeyTimer       *time.Timer
	isScrolling       bool
	gowid.IsSelectable
}

func New(defaultID, lastID string) (*Widget, error) {
	var term string
	for _, s := range os.Environ() {
		if strings.HasPrefix(s, "TERM=") {
			term = s[len("TERM="):]
			break
		}
	}

	var (
		ti  *terminfo.Terminfo
		err error
	)
	if term != "" {
		ti, err = findTerminfo(term)
	}
	if ti == nil {
		// Fallback to xterm.
		ti, err = findTerminfo("xterm")
	}
	if err != nil {
		return nil, err
	}

	return &Widget{
		IHotKeyProvider:    gowidterminal.HotKey{tcell.KeyCtrlB},
		IHotKeyPersistence: gowidterminal.HotKeyDuration{time.Second},
		Callbacks:          gowid.NewCallbacks(),
		terminfo:           ti,
		defaultID:          defaultID,
		lastID:             lastID,
	}, nil
}

func (w *Widget) String() string {
	return "terminal"
}

func (w *Widget) Write(p []byte) (n int, err error) {
	return w.vt.Write(p)
}

func (w *Widget) MouseSupport() gowidterminal.IMouseSupport {
	return &mouseSupport{w.vt.Mode()}
}

func (w *Widget) Terminfo() *terminfo.Terminfo {
	return w.terminfo
}

func (w *Widget) HotKeyActive() bool {
	return w.hotKeyDown
}

func (w *Widget) SetHotKeyActive(app gowid.IApp, down bool) {
	w.hotKeyDown = down

	if w.hotKeyTimer != nil {
		w.hotKeyTimer.Stop()
	}

	if down {
		w.hotKeyDownTime = time.Now()
		w.hotKeyTimer = time.AfterFunc(w.HotKeyDuration(), func() {
			app.Run(gowid.RunFunction(func(app gowid.IApp) {
				w.SetHotKeyActive(app, false)
			}))
		})
	}
}

func (w *Widget) HotKeyDownTime() time.Time {
	return w.hotKeyDownTime
}

func (w *Widget) Scroll(dir ScrollDir, page bool, lines int) {
	if page {
		lines = w.canvas.ScrollBuffer(dir, false, gwutil.NoneInt())
	} else {
		lines = w.canvas.ScrollBuffer(dir, false, gwutil.SomeInt(lines))
	}
	w.isScrolling = true
}

func (w *Widget) ResetScroll() {
	w.isScrolling = false
	w.canvas.ScrollBuffer(ScrollNone, true, gwutil.NoneInt())
}

func (w *Widget) Scrolling() bool {
	return w.isScrolling
}

func (w *Widget) OnTitleChanged(f gowid.IWidgetChangedCallback) {
	gowid.AddWidgetCallback(w.Callbacks, TitleChanged{}, f)
}

func (w *Widget) OnProcessExited(f gowid.IWidgetChangedCallback) {
	gowid.AddWidgetCallback(w.Callbacks, ProcessExited{}, f)
}

func (w *Widget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	box, ok := size.(gowid.IRenderBox)
	if !ok {
		panic(gowid.WidgetSizeError{Widget: w, Size: size, Required: "gowid.IRenderBox"})
	}

	w.TouchTerminal(box.BoxColumns(), box.BoxRows(), app)
	return w.canvas
}

func (w *Widget) RenderSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	box, ok := size.(gowid.IRenderBox)
	if !ok {
		panic(gowid.WidgetSizeError{Widget: w, Size: size, Required: "gowid.IRenderBox"})
	}
	return gowid.RenderBox{C: box.BoxColumns(), R: box.BoxRows()}
}

func (w *Widget) TouchTerminal(width, height int, app gowid.IApp) {
	if w.Canvas() == nil {
		w.canvas = NewCanvasOfSize(width, height)
	}

	setTermSize := false
	if !w.Connected() {
		var err error
		w.vt, err = vt.New(width, height)
		if err != nil {
			panic(err)
		}

		renderCh := make(chan string, 1)
		w.vt.Subscribe("host", renderCh)

		go func() {
			for {
				select {
				case <-w.vt.Done():
					app.Run(gowid.RunFunction(func(app gowid.IApp) {
						gowid.RunWidgetCallbacks(w.Callbacks, ProcessExited{}, app, w, w.lastID)
					}))
					return
				case <-renderCh:
					app.Run(gowid.RunFunction(func(runApp gowid.IApp) {
						w.vt.Lock()
						cols, rows := w.vt.Size()
						w.RenderTerminal(cols, rows, app)
						w.vt.Unlock()
						runApp.Redraw()
					}))

					title := w.vt.Title()
					if title != w.title {
						w.title = title
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							gowid.RunWidgetCallbacks(w.Callbacks, TitleChanged{}, app, w, title)
						}))
					}
				}
			}
		}()

		setTermSize = true
	}

	if !(w.width == width && w.height == height) {
		if !setTermSize {
			w.vt.Resize(width, height)
		}

		w.Canvas().Resize(width, height)
		w.width = width
		w.height = height
	}

	w.vt.Lock()
	cols, rows := w.vt.Size()
	w.RenderTerminal(cols, rows, app)
	w.vt.Unlock()
}

func (w *Widget) RenderTerminal(cols, rows int, app gowid.IApp) {
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			glyph := w.vt.Cell(x, y)

			fg, bg := int(glyph.FG), int(glyph.BG)
			if glyph.FG == vt10x.DefaultFG {
				fg = int(tcell.ColorDefault)
			}
			if glyph.BG == vt10x.DefaultBG {
				bg = int(tcell.ColorDefault)
			}

			cell := gowid.MakeCell(glyph.Char, getColor256(fg), getColor256(bg), gowid.StyleNone)
			w.canvas.SetCellAt(x, y, cell)
		}
	}

	if w.Scrolling() {
		text := fmt.Sprintf("[%d/%d]", w.canvas.Offset, w.canvas.Canvas.BoxRows()-w.canvas.BoxRows())
		for i := 0; i < len(text); i++ {
			x := cols - len(text) + i
			if x < 0 || x >= cols {
				continue
			}
			cell := gowid.MakeCell(rune(text[i]), gowid.ColorBlack, gowid.ColorYellow, gowid.StyleNone)
			w.canvas.SetCellAt(x, 0, cell)
		}
	}

	p := app.(wid.IP2PApp)
	id, palette := p.FocusPalette(w.lastID)
	if palette != nil && w.vt.CursorVisible() {
		f, _, _ := palette.GetStyle(app)
		paletteFG, _ := f.ToTCellColor(app.GetColorMode())

		cursor := w.vt.Cursor()
		if cursor.X > 0 && cursor.X < cols && cursor.Y > 0 && cursor.Y < rows {
			glyph := w.vt.Cell(cursor.X, cursor.Y)

			bg := int(glyph.BG)
			if glyph.BG == vt10x.DefaultBG {
				bg = int(tcell.ColorDefault)
			}

			var cell gowid.Cell
			if glyph.Char == ' ' {
				cell = gowid.MakeCell('⎸', paletteFG, getColor256(bg), gowid.StyleBold)
			} else {
				cell = gowid.MakeCell(glyph.Char, paletteFG, getColor256(bg), gowid.StyleReverse)
			}
			w.canvas.SetCellAt(cursor.X, cursor.Y, cell)

			text := id
			if len(text) > 6 {
				text = text[:6]
			}
			for i := 0; i < len(text); i++ {
				if cursor.X+i >= cols || cursor.Y+1 >= rows {
					continue
				}
				cell = gowid.MakeCell(rune(text[i]), gowid.ColorWhite, paletteFG, gowid.StyleNone)
				w.canvas.SetCellAt(cursor.X+i, cursor.Y+1, cell)
			}
		}
	}
}

func (w *Widget) Canvas() *Canvas {
	return w.canvas
}

func (w *Widget) Connected() bool {
	return w.vt != nil
}

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	handled := false
	// True if input should be sent to tty.
	passToTty := true

	evt := ev
	id := w.defaultID
	if evr, ok := ev.(*rvt.RemoteEvent); ok {
		id = evr.ID
		evt = evr.Event
	}
	if evk, ok := evt.(*tcell.EventKey); ok {
		// if w.Scrolling() {
		// 	// If we're currently scrolling, then this user input should
		// 	// never be sent to the terminal. It's for controlling or exiting
		// 	// scrolling.
		// 	passToTty = false
		// 	handled = true
		// 	switch evk.Key() {
		// 	case tcell.KeyPgUp:
		// 		w.Scroll(ScrollUp, true, 0)
		// 	case tcell.KeyPgDn:
		// 		w.Scroll(ScrollDown, true, 0)
		// 	case tcell.KeyUp:
		// 		w.Scroll(ScrollUp, false, 1)
		// 	case tcell.KeyDown:
		// 		w.Scroll(ScrollDown, false, 1)
		// 	case tcell.KeyEnter:
		// 		w.ResetScroll()
		// 	case tcell.KeyRune:
		// 		switch evk.Rune() {
		// 		case 'q', 'Q':
		// 			w.ResetScroll()
		// 		}
		// 	default:
		// 		handled = false
		// 	}
		if w.HotKeyActive() {
			passToTty = false
			deactivate := false
			switch evk.Key() {
			case w.HotKey():
				handled = true
				deactivate = true
			case tcell.KeyRune:
				// switch evk.Rune() {
				// case '[':
				// 	w.Scroll(ScrollUp, false, 0)
				// 	handled = true
				// 	deactivate = true
				// }
			}
			if deactivate {
				w.SetHotKeyActive(app, false)
			}
		} else if evk.Key() == w.HotKey() {
			passToTty = false
			w.SetHotKeyActive(app, true)
			handled = true
		}
	}

	if !handled {
		if evm, ok := evt.(*tcell.EventMouse); ok {
			ss := w.RenderSize(size, focus, app)
			x, y := evm.Position()
			if !(x < ss.BoxColumns()) && (y < ss.BoxRows()) {
				passToTty = false
			}
		}
	}
	if passToTty {
		seq, parsed := gowidterminal.TCellEventToBytes(evt, w.MouseSupport(), app.GetLastMouseState(), w.Terminfo())
		if parsed {
			_, err := w.Write(seq)
			if err != nil {
				panic(err)
				// todo get context for logging
			}
			handled = true
			w.lastID = id
		}
	}
	return handled
}

// getColor256 returns the tcell color for a number between 0 and 255
func getColor256(color int) gowid.TCellColor {
	if color == 0 {
		return gowid.ColorDefault
	}
	return gowid.MakeTCellColorExt(tcell.PaletteColor(color))
}
