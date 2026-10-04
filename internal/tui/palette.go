package tui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

// palette is every style the screen draws with (reap:943-987). A terminal
// that cannot say whether its background is light or dark gets no fills.
// One with fewer than 256 colours gets plain ANSI colours and attributes.
type palette struct {
	live, ok, yel, danger, acc, dim, line, sel, warn tcell.Style
	tabOn, trashOn, tabOff, tabCnt, tabPin, label    tcell.Style
	bandT, bandD, bandAcc, bandRule                  tcell.Style
	tint, tintLive, tintAcc                          tcell.Style
	trayD, trayLive, trayAcc                         tcell.Style
	bold                                             tcell.Style
	// divider is the line colour, or dim where the terminal has none.
	divider tcell.Style
	// band is true when tabs and group headers sit on a tinted band, tray
	// when hidden chats sit on a darker one.
	band, tray bool
}

// newPalette builds the look for a terminal with this many colours. light and
// known are the answer of lightBackground.
func newPalette(colours int, light, known bool) palette {
	plain := tcell.StyleDefault
	fg := func(n int) tcell.Style { return plain.Foreground(color.PaletteColor(n)) }
	both := func(f, b int) tcell.Style { return fg(f).Background(color.PaletteColor(b)) }
	var p palette
	p.bold = plain.Bold(true)
	if colours >= 256 {
		p.live = fg(71)
		p.yel = fg(136)
		p.danger = fg(131).Bold(true)
		p.acc = fg(67)
		p.dim = fg(245)
		p.line = fg(245)
		if known {
			p.line = fg(240)
			if light {
				p.line = fg(247)
			}
		}
		p.sel = both(231, 60).Bold(true)
		p.tabOn = both(231, 67).Bold(true)
		p.trashOn = both(231, 136).Bold(true)
		p.bandRule = fg(250)
		p.divider = p.line
		if known {
			band, tint, tray, tintAcc := 236, 60, 235, 153
			if light {
				band, tint, tray, tintAcc = 254, 189, 255, 67
			}
			p.band, p.tray = true, true
			p.bandT, p.bandD, p.bandAcc = plain.Background(color.PaletteColor(band)), both(245, band), both(67, band).Bold(true)
			p.tint, p.tintLive, p.tintAcc = plain.Background(color.PaletteColor(tint)), both(71, tint), both(tintAcc, tint).Bold(true)
			p.trayD, p.trayLive, p.trayAcc = both(245, tray), both(71, tray), both(67, tray).Bold(true)
		}
	} else {
		p.live = fg(2)
		p.yel = fg(3)
		p.danger = fg(1).Bold(true)
		p.acc = fg(6)
		p.dim = plain.Dim(true)
		p.bandRule = p.dim
		p.sel = plain.Reverse(true).Bold(true)
		p.tabOn = both(0, 6).Bold(true)
		p.trashOn = both(0, 3).Bold(true)
		p.divider = p.dim
	}
	p.ok = p.live.Bold(true)
	p.warn = p.yel.Bold(true)
	p.tabOff, p.tabCnt, p.tabPin = p.dim, p.dim, p.acc
	p.label = p.bold
	if p.band {
		p.tabOff, p.tabCnt, p.tabPin = p.bandT, p.bandD, p.bandAcc
		p.label = p.bandT.Bold(true)
	}
	return p
}
