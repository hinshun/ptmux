package ui

type Boundary string

const (
	TopRight    Boundary = "┐"
	Vertical    Boundary = "│"
	Horizontal  Boundary = "─"
	TopLeft     Boundary = "┌"
	BottomRight Boundary = "┘"
	BottomLeft  Boundary = "└"
)
