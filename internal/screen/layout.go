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
// CursorX/Y are initialised to the centre of the server screen so that
// edge detection fires after roughly half a screen-width of movement,
// not a full screen-width from the top-left corner.
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
			l.ActiveID--
			prev := l.screenByID(l.ActiveID)
			l.CursorX = prev.Width / 2
			return l.ActiveID, true
		}
		l.CursorX = 0
	}

	if l.CursorX >= active.Width {
		if int(l.ActiveID) < len(l.Screens)-1 {
			l.ActiveID++
			next := l.screenByID(l.ActiveID)
			l.CursorX = next.Width / 2
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

// RevertSwitch undoes a switch that UpdateCursor just made when the target
// screen turned out to have no connected client. prevID is the screen we
// were on before; movedRight tells us which edge to clamp to.
func (l *Layout) RevertSwitch(prevID uint8, movedRight bool) {
	l.ActiveID = prevID
	prev := l.screenByID(prevID)
	// Reset to center so the user never needs more than Width/2 movement to switch back.
	l.CursorX = prev.Width / 2
	_ = movedRight
}

func (l *Layout) screenByID(id uint8) Screen {
	for _, s := range l.Screens {
		if s.ID == id {
			return s
		}
	}
	return l.Screens[0]
}
