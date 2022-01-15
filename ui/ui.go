package ui

import (
	"context"
	"unicode"
	"unicode/utf8"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/vt10x"
)

type UI struct {
	view          vt10x.View
	screen        tcell.Screen
	width, height int
}

func New(view vt10x.View) (*UI, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	err = screen.Init()
	if err != nil {
		return nil, err
	}

	width, height := screen.Size()
	height--
	view.Resize(width, height)

	return &UI{
		view:   view,
		screen: screen,
		width:  width,
		height: height,
	}, nil
}

func (u *UI) Loop(ctx context.Context) error {
	eventCh := make(chan tcell.Event, 4)
	go func() {
		defer close(eventCh)
		for {
			event := u.screen.PollEvent()
			if event == nil {
				return
			}
			eventCh <- event
		}
	}()

	for {
		select {
		case <-ctx.Done():
			u.screen.Fini()
			// Drain remaining events.
			for _ = range eventCh {
			}
			return nil
		case event := <-eventCh:
			switch ev := event.(type) {
			case *tcell.EventResize:
				u.width, u.height = ev.Size()
				u.height--
				u.view.Resize(u.width, u.height)
				u.screen.Sync()
			}
		}
	}
}

func (u *UI) Render(title string, cols, rows int) {
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			glyph := u.view.Cell(x, y)

			fg, bg := int(glyph.FG), int(glyph.BG)
			if glyph.FG == vt10x.DefaultFG {
				fg = int(tcell.ColorDefault)
			}
			if glyph.BG == vt10x.DefaultBG {
				bg = int(tcell.ColorDefault)
			}

			style := tcell.StyleDefault.
				Foreground(getColor256(fg)).
				Background(getColor256(bg))
			u.screen.SetContent(x, y, glyph.Char, nil, style)
		}
	}

	text := []byte(title)
	textLen := characterCount(text)
	for x := 0; x < cols; x++ {
		style := tcell.StyleDefault.Reverse(true)
		if x < textLen {
			ch, combc, size := decodeCharacter(text)
			text = text[size:]
			u.screen.SetContent(x, rows, ch, combc, style)
		} else {
			u.screen.SetContent(x, rows, ' ', nil, style)
		}
	}
	if u.view.CursorVisible() {
		cursor := u.view.Cursor()
		u.screen.ShowCursor(cursor.X, cursor.Y)
	} else {
		u.screen.HideCursor()
	}
	u.screen.Show()
}

// getColor256 returns the tcell color for a number between 0 and 255
func getColor256(color int) tcell.Color {
	if color == 0 {
		return tcell.ColorDefault
	}
	return tcell.PaletteColor(color)
}

// characterCount returns the number of characters in a byte array
// Similar to utf8.RuneCount but for unicode characters
func characterCount(b []byte) int {
	s := 0

	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if !isMark(r) {
			s++
		}

		b = b[size:]
	}

	return s
}

func decodeCharacter(b []byte) (rune, []rune, int) {
	r, size := utf8.DecodeRune(b)
	b = b[size:]
	c, s := utf8.DecodeRune(b)

	var combc []rune
	for isMark(c) {
		combc = append(combc, c)
		size += s

		b = b[s:]
		c, s = utf8.DecodeRune(b)
	}

	return r, combc, size
}

var minMark = rune(unicode.Mark.R16[0].Lo)

func isMark(r rune) bool {
	// Fast path
	if r < minMark {
		return false
	}
	return unicode.In(r, unicode.Mark)
}
