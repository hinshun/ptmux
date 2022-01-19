package terminal

import (
	"os"
	"strings"
	"time"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/null"
	gowidterminal "github.com/gcla/gowid/widgets/terminal"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/terminfo"
)

type IWidget interface {
	gowid.IWidget
	gowidterminal.IHotKeyProvider
	gowidterminal.IHotKeyPersistence
	HotKeyActive() bool
	SetHotKeyActive(app gowid.IApp, down bool)
	HotKeyDownTime() time.Time
}

type Widget struct {
	gowid.IWidget
	gowidterminal.IHotKeyProvider
	gowidterminal.IHotKeyPersistence
	gowid.IsSelectable
	terminfo       *terminfo.Terminfo
	hotKeyDown     bool
	hotKeyDownTime time.Time
	hotKeyTimer    *time.Timer
}

type Options struct {
	Env                []string
	HotKey             gowidterminal.IHotKeyProvider
	HotKeyPersistence gowidterminal.IHotKeyPersistence
}

func New() (*Widget, error) {
	return NewExt(Options{
		Env: os.Environ(),
	})
}

func NewExt(opts Options) (*Widget, error) {
	var term string
	for _, s := range opts.Env {
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

	if opts.HotKey == nil {
		opts.HotKey = gowidterminal.HotKey{tcell.KeyCtrlB}
	}

	if opts.HotKeyPersistence == nil {
		opts.HotKeyPersistence = gowidterminal.HotKeyDuration{time.Second * 3}
	}

	return &Widget{
		IWidget:            null.New(),
		IHotKeyProvider:    opts.HotKey,
		IHotKeyPersistence: opts.HotKeyPersistence,
		terminfo:           ti,
	}, nil
}

func (w *Widget) Terminfo() *terminfo.Terminfo {
	return w.terminfo
}

func (w *Widget) Selectable() bool {
	return true
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

func (w *Widget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	return UserInput(w, ev, size, focus, app)
}

func UserInput(w IWidget, ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	handled := false
	// True if input should be sent to tty.
	passToTty := true

	if evk, ok := ev.(*tcell.EventKey); ok {
		if w.HotKeyActive() {
			passToTty = false
			handled = true
			deactivate := false
			switch evk.Key() {
			case w.HotKey():
				deactivate = true
			default:
				handled = false
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
		if evm, ok := ev.(*tcell.EventMouse); ok {
			ss := w.RenderSize(size, focus, app)
			x, y := evm.Position()
			if !(x < ss.BoxColumns()) && (y < ss.BoxRows()) {
				passToTty = false
			}
		}
	}
	if passToTty {
		// seq, parsed := gowidterminal.TCellEventToBytes(ev, w.Modes(), app.GetLastMouseState(), w.Terminfo())
		// if parsed {
		// 	_, err := w.Write(seq)
		// 	if err != nil {
		// 		// todo get context for logging
		// 	}
		// 	handled = true
		// }
	}
	return handled
}
