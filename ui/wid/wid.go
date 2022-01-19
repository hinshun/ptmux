package wid

import (
	"sort"

	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
)

const (
	DefaultID = "self"
)

type IFocus interface {
	Focus(id string) int
	SetFocus(id string, i int)
}

type ICompositeMultipleFocus interface {
	gowid.ICompositeMultiple
	IFocus
}

type IP2PApp interface {
	IDs() []string
	FocusPalette(id string) gowid.ICellStyler
	SetClickTarget(k tcell.ButtonMask, w gowid.IIdentityWidget) bool
	ClickTarget(func(tcell.ButtonMask, gowid.IIdentityWidget))
	GetMouseState() gowid.MouseState
	GetLastMouseState() gowid.MouseState
}

type app struct {
	gowid.IApp
	ids            []string
	palette        map[string]gowid.ICellStyler
	clickTargets   gowid.ClickTargets
	mouseState     gowid.MouseState
	lastMouseState gowid.MouseState
}

func WithP2PContext(a gowid.IApp, palette map[string]gowid.ICellStyler, clickTargets gowid.ClickTargets, mouseState, lastMouseState gowid.MouseState) gowid.IApp {
	var ids []string
	for id := range palette {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return &app{
		IApp:           a,
		ids:            ids,
		palette:        palette,
		clickTargets:   clickTargets,
		mouseState:     mouseState,
		lastMouseState: lastMouseState,
	}
}

func WithFocus(a gowid.IApp, ids []string) gowid.IApp {
	fa, ok := a.(*app)
	if !ok {
		fa.IApp = a
	}
	ids = FilterFocus(fa.ids, ids)
	return &app{
		IApp:    fa.IApp,
		ids:     ids,
		palette: fa.palette,
		clickTargets: fa.clickTargets,
		mouseState: fa.mouseState,
		lastMouseState: fa.lastMouseState,
	}
}

func (a *app) IDs() []string {
	sort.Strings(a.ids)
	return a.ids
}

func (a *app) FocusPalette(id string) gowid.ICellStyler {
	if len(a.ids) == 0 {
		return nil
	}
	sort.Strings(a.ids)
	return a.palette[id]
}

func (a *app) SetClickTarget(k tcell.ButtonMask, w gowid.IIdentityWidget) bool {
	return a.clickTargets.SetClickTarget(k, w)
}

func (a *app) ClickTarget(f func(tcell.ButtonMask, gowid.IIdentityWidget)) {
	a.clickTargets.ClickTarget(f)
}

func (a *app) GetMouseState() gowid.MouseState {
	return a.mouseState
}

func (a *app) GetLastMouseState() gowid.MouseState {
	return a.lastMouseState
}

func FilterFocus(parent, child []string) []string {
	parentSet := make(map[string]struct{})
	for _, id := range parent {
		parentSet[id] = struct{}{}
	}

	var focus []string
	for _, id := range child {
		if _, ok := parentSet[id]; ok {
			focus = append(focus, id)
		}
	}
	return focus
}
