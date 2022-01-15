package rvt

import (
	"sync"

	"github.com/hinshun/vt10x"
)

type View struct {
	mu         sync.Mutex
	cols, rows int
	mode       vt10x.ModeFlag
	title      string
	lines      [][]vt10x.Glyph
	cursor     vt10x.Cursor
}

func NewView() *View {
	return &View{}
}

func (v *View) Update(state *StateMessage) {
	v.cols = int(state.Cols)
	v.rows = int(state.Rows)
	v.mode = vt10x.ModeFlag(state.Mode)
	v.title = state.Title

	var lines [][]vt10x.Glyph
	for _, l := range state.Lines {
		var glyphs []vt10x.Glyph
		for _, g := range l.Glyphs {
			glyphs = append(glyphs, protoGlyphToVTGlyph(g))
		}
		lines = append(lines, glyphs)
	}
	v.lines = lines

	v.cursor = vt10x.Cursor{
		Attr:  protoGlyphToVTGlyph(state.Cursor.Attr),
		X:     int(state.Cursor.X),
		Y:     int(state.Cursor.Y),
		State: uint8(state.Cursor.State),
	}
}

func (v *View) String() string {
	v.Lock()
	defer v.Unlock()

	var view []rune
	for y := 0; y < v.rows; y++ {
		for x := 0; x < v.cols; x++ {
			attr := v.Cell(x, y)
			view = append(view, attr.Char)
		}
		view = append(view, '\n')
	}

	return string(view)
}

func (v *View) Resize(cols, rows int) {
	// TODO:
}

// Size returns the size of the virtual terminal.
func (v *View) Size() (cols, rows int) {
	return v.cols, v.rows
}

// Mode returns the current terminal mode.//
func (v *View) Mode() vt10x.ModeFlag {
	return v.mode
}

// Title represents the title of the console window.
func (v *View) Title() string {
	return v.title
}

// Cell returns the glyph containing the character code, foreground color, and
// background color at position (x, y) relative to the top left of the terminal.
func (v *View) Cell(x, y int) vt10x.Glyph {
	return v.lines[y][x]
}

// Cursor returns the current position of the cursor.
func (v *View) Cursor() vt10x.Cursor {
	return v.cursor
}

// CursorVisible returns the visible state of the cursor.
func (v *View) CursorVisible() bool {
	return v.mode&vt10x.ModeHide == 0
}

// Lock locks the state object's mutex.
func (v *View) Lock() {
	v.mu.Lock()
}

// Unlock resets change flags and unlocks the state object's mutex.
func (v *View) Unlock() {
	v.mu.Unlock()
}
