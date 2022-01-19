package terminal

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	gowidterminal "github.com/gcla/gowid/widgets/terminal"
)

type ScrollDir int

const (
	ScrollNone ScrollDir = iota
	ScrollDown
	ScrollUp
)

type Canvas struct {
	*gowidterminal.ViewPortCanvas
}

func NewCanvasOfSize(cols, rows int) *Canvas {
	c := gowid.NewCanvasOfSize(cols, rows)
	return &Canvas{
		ViewPortCanvas: gowidterminal.NewViewPort(c, 0, rows),
	}
}

func (c *Canvas) Resize(width, height int) {
	if width > c.BoxColumns() {
		c.ExtendRight(gowid.EmptyLine(width - c.BoxColumns()))
	} else if width < c.BoxColumns() {
		c.TrimRight(width)
	}

	c.Offset -= height - c.Height
	c.Height = height
	if c.Height > c.Canvas.BoxRows() {
		c.Canvas.AppendBelow(gowid.NewCanvasOfSize(width, c.Height-c.Canvas.BoxRows()), false, false)
	} else if c.Height < 1 {
		c.Height = 1
	}
	if c.Offset < 0 {
		c.Offset = 0
	} else if c.Offset > (c.Canvas.BoxRows() - c.Height) {
		c.Offset = c.Canvas.BoxRows() - c.Height
	}
}

func (c *Canvas) ScrollBuffer(dir ScrollDir, reset bool, linesOpt gwutil.IntOption) int {
	prev := c.Offset
	if reset {
		c.Offset = c.Canvas.BoxRows() - c.BoxRows()
	} else {
		var lines int
		if linesOpt.IsNone() {
			lines = c.BoxRows() / 2
		} else {
			lines = linesOpt.Val()
		}
		if dir == ScrollDown {
			lines = -lines
		}
		maxScroll := c.Canvas.BoxRows() - c.BoxRows()
		c.Offset -= lines
		if c.Offset < 0 {
			c.Offset = 0
		} else if c.Offset > maxScroll {
			c.Offset = maxScroll
		}
	}
	return c.Offset - prev
}
