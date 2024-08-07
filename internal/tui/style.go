package tui

import "fmt"

type StyleProperty int

const (
	NoStyle       = StyleProperty(iota)
	PropertyTrue  = StyleProperty(iota + 1)
	PropertyFalse = StyleProperty(iota + 2)
)

type Color struct {
	RGB [3]int
}

var (
	NoColor = Color{RGB: [3]int{-1, -1, -1}}
)

type Style struct {
	Foreground Color
	Background Color
	Bold       StyleProperty
	Underline  StyleProperty
	Italic     StyleProperty
}

func NewColor(r int, g int, b int) Color {
	var c Color
	c.RGB[0] = r
	c.RGB[1] = g
	c.RGB[2] = b
	return c
}

func NewStyle() Style {
	return Style{
		Foreground: NoColor,
		Background: NoColor,
	}
}

func applyStyle(style *Style) string {
	res := ""

	if style == nil {
		return "\033[0m"
	}

	if style.Foreground != NoColor {
		res += fmt.Sprintf("\033[38;2;%d;%d;%dm", style.Foreground.RGB[0]&0xff, style.Foreground.RGB[1]&0xff, style.Foreground.RGB[2]&0xff)
	} else {
		res += "\033[39m"
	}

	if style.Background != NoColor {
		res += fmt.Sprintf("\033[48;2;%d;%d;%dm", style.Background.RGB[0]&0xff, style.Background.RGB[1]&0xff, style.Background.RGB[2]&0xff)
	} else {
		res += "\033[49m"
	}

	if style.Bold == PropertyTrue {
		res += "\033[1m"
	} else {
		res += "\033[22m"
	}

	if style.Italic != PropertyTrue {
		res += "\033[3m"
	} else {
		res += "\033[23m"
	}

	if style.Underline != PropertyTrue {
		res += "\033[4m"
	} else {
		res += "\033[24m"
	}

	return res
}
