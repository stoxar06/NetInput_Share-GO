// Package screen manages the logical layout of 4 screens and handles
// cursor edge detection to trigger screen switching.
package screen

import "fmt"

// EntryX is the x-coordinate a client cursor is warped to when entering from
// the left (coming from the previous screen). Must match network/server.go's
// entryX constant so the server's CursorX tracking starts at the same value.
const EntryX = 5

// Screen represents one laptop in the layout.
type Screen struct {
	ID     uint8
	Name   string
	IP     string
	Width  int
	Height int
}

// Layout tracks the logical cursor position across all screens.
// CursorX is an absolute x-coordinate within the active screen.
// When a client screen is entered the server sends a warp so the visual
// cursor matches CursorX, making edge detection accurate.
type Layout struct {
	Screens  []Screen
	Mode     string
	ActiveID uint8
	CursorX  int
	CursorY  int
}

// NewLayout creates a Layout. CursorX starts at the centre of screen 0 —
// a reasonable approximation since we don't warp the server cursor at start.
func NewLayout(screens []Screen, mode string) *Layout {
	l := &Layout{Screens: screens, Mode: mode}
	if len(screens) > 0 {
		l.CursorX = screens[0].Width / 2
		l.CursorY = screens[0].Height / 2
	}
	return l
}

// ActiveScreen returns the currently focused screen.
func (l *Layout) ActiveScreen() Screen {
	for _, s := range l.Screens {
		if s.ID == l.ActiveID {
			return s
		}
	}
	return l.Screens[0]
}

// UpdateCursor moves the cursor by (dx, dy) and returns the new active screen
// ID plus whether a switch occurred.
//
// Entering from the left (switching right): CursorX = EntryX.
// The server simultaneously sends PacketWarpCursor(EntryX, Height/2) to the
// client so its visual cursor matches — edge detection is then accurate.
//
// Entering from the right (switching back): CursorX = Width/2 (centre).
// No warp is sent to the server screen; the user presses right ~Width/2 px
// to switch again, or uses Ctrl+Alt+→.
func (l *Layout) UpdateCursor(dx, dy int) (newID uint8, switched bool) {
	active := l.ActiveScreen()

	l.CursorX += dx
	l.CursorY += dy
	if l.CursorY < 0 {
		l.CursorY = 0
	}
	if l.CursorY >= active.Height {
		l.CursorY = active.Height - 1
	}

	// Past left edge → switch to previous screen.
	if l.CursorX < 0 {
		if l.ActiveID > 0 {
			l.ActiveID--
			prev := l.screenByID(l.ActiveID)
			l.CursorX = prev.Width / 2 // enter previous screen at centre
			return l.ActiveID, true
		}
		l.CursorX = 0 // already at leftmost screen
	}

	// Past right edge → switch to next screen.
	if l.CursorX >= active.Width {
		if int(l.ActiveID) < len(l.Screens)-1 {
			l.ActiveID++
			l.CursorX = EntryX // enter new screen at known entry point (matches warp)
			return l.ActiveID, true
		}
		l.CursorX = active.Width - 1 // already at rightmost screen
	}

	return l.ActiveID, false
}

// SwitchNext moves focus to the next screen (hotkey: Ctrl+Alt+→).
func (l *Layout) SwitchNext() (uint8, error) {
	if int(l.ActiveID) >= len(l.Screens)-1 {
		return l.ActiveID, fmt.Errorf("already at last screen")
	}
	l.ActiveID++
	l.CursorX = EntryX
	return l.ActiveID, nil
}

// SwitchPrev moves focus to the previous screen (hotkey: Ctrl+Alt+←).
func (l *Layout) SwitchPrev() (uint8, error) {
	if l.ActiveID == 0 {
		return l.ActiveID, fmt.Errorf("already at first screen")
	}
	l.ActiveID--
	prev := l.screenByID(l.ActiveID)
	l.CursorX = prev.Width / 2
	return l.ActiveID, nil
}

// RevertSwitch undoes a switch when the target screen has no connected client.
func (l *Layout) RevertSwitch(prevID uint8, movedRight bool) {
	l.ActiveID = prevID
	prev := l.screenByID(prevID)
	if movedRight {
		l.CursorX = prev.Width - 1 // clamp at right edge
	} else {
		l.CursorX = 0 // clamp at left edge
	}
}

func (l *Layout) screenByID(id uint8) Screen {
	for _, s := range l.Screens {
		if s.ID == id {
			return s
		}
	}
	return l.Screens[0]
}
