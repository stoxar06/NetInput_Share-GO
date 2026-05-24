// Package screen manages the logical layout of 4 screens and handles
// cursor edge detection to trigger screen switching.
package screen

import "fmt"

// Screen represents one laptop in the layout.
type Screen struct {
	ID     uint8
	Name   string
	IP     string
	Width  int
	Height int
}

// Layout holds the arrangement of screens.
type Layout struct {
	Screens  []Screen
	Mode     string
	ActiveID uint8
	CursorX  int
	CursorY  int
}

// NewLayout creates a Layout from config screens.
func NewLayout(screens []Screen, mode string) *Layout {
	return &Layout{Screens: screens, Mode: mode}
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

// UpdateCursor moves the logical cursor by dx,dy and returns the new active
// screen ID plus whether a screen switch occurred.
func (l *Layout) UpdateCursor(dx, dy int) (newID uint8, switched bool) {
	l.CursorX += dx
	l.CursorY += dy

	active := l.ActiveScreen()

	if l.CursorY < 0 {
		l.CursorY = 0
	}
	if l.CursorY >= active.Height {
		l.CursorY = active.Height - 1
	}

	if l.CursorX < 0 {
		if l.ActiveID > 0 {
			left := l.screenByID(l.ActiveID - 1)
			l.ActiveID--
			l.CursorX = left.Width - 1
			return l.ActiveID, true
		}
		l.CursorX = 0
	}

	if l.CursorX >= active.Width {
		if int(l.ActiveID) < len(l.Screens)-1 {
			l.ActiveID++
			l.CursorX = 0
			return l.ActiveID, true
		}
		l.CursorX = active.Width - 1
	}

	return l.ActiveID, false
}

// SwitchNext moves focus to the next screen (hotkey: Ctrl+Alt+→).
func (l *Layout) SwitchNext() (uint8, error) {
	if int(l.ActiveID) >= len(l.Screens)-1 {
		return l.ActiveID, fmt.Errorf("already at last screen")
	}
	l.ActiveID++
	return l.ActiveID, nil
}

// SwitchPrev moves focus to the previous screen (hotkey: Ctrl+Alt+←).
func (l *Layout) SwitchPrev() (uint8, error) {
	if l.ActiveID == 0 {
		return l.ActiveID, fmt.Errorf("already at first screen")
	}
	l.ActiveID--
	return l.ActiveID, nil
}

func (l *Layout) screenByID(id uint8) Screen {
	for _, s := range l.Screens {
		if s.ID == id {
			return s
		}
	}
	return l.Screens[0]
}
