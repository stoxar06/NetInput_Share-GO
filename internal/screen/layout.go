// Package screen manages the logical layout of 4 screens and handles
// cursor edge detection to trigger screen switching.
package screen

import "fmt"

// switchThreshold is how many pixels the mouse must travel continuously in
// one direction before a screen switch fires. Resets to zero on any reversal,
// so small wobbles never trigger an accidental switch.
const switchThreshold = 300

// Screen represents one laptop in the layout.
type Screen struct {
	ID     uint8
	Name   string
	IP     string
	Width  int
	Height int
}

// Layout holds the arrangement of screens.
// CursorX is a directional push buffer (not an absolute position):
// positive = rightward push accumulated, negative = leftward push accumulated.
// It resets to zero whenever the mouse reverses direction.
type Layout struct {
	Screens  []Screen
	Mode     string
	ActiveID uint8
	CursorX  int // directional push buffer
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

// UpdateCursor processes one mouse-move delta and returns the new active
// screen ID plus whether a screen switch occurred.
//
// CursorX is a directional push buffer: it accumulates movement in one
// direction and resets to zero when the direction reverses. A switch fires
// only after switchThreshold pixels of uninterrupted movement past the edge.
// This means the cursor can move freely on any screen without accidentally
// triggering a switch — only a deliberate sustained sweep does it.
func (l *Layout) UpdateCursor(dx, dy int) (newID uint8, switched bool) {
	// Y clamping (cosmetic only).
	active := l.ActiveScreen()
	l.CursorY += dy
	if l.CursorY < 0 {
		l.CursorY = 0
	}
	if l.CursorY >= active.Height {
		l.CursorY = active.Height - 1
	}

	// Directional push buffer: reset on direction reversal.
	if dx > 0 {
		if l.CursorX < 0 {
			l.CursorX = 0
		}
		l.CursorX += dx
	} else if dx < 0 {
		if l.CursorX > 0 {
			l.CursorX = 0
		}
		l.CursorX += dx
	}

	// Right push: switch to next screen.
	if l.CursorX >= switchThreshold {
		if int(l.ActiveID) < len(l.Screens)-1 {
			l.ActiveID++
			l.CursorX = 0
			return l.ActiveID, true
		}
		l.CursorX = switchThreshold // cap; no next screen
	}

	// Left push: switch to previous screen.
	if l.CursorX <= -switchThreshold {
		if l.ActiveID > 0 {
			l.ActiveID--
			l.CursorX = 0
			return l.ActiveID, true
		}
		l.CursorX = -switchThreshold // cap; already at first screen
	}

	return l.ActiveID, false
}

// SwitchNext moves focus to the next screen (hotkey: Ctrl+Alt+→).
func (l *Layout) SwitchNext() (uint8, error) {
	if int(l.ActiveID) >= len(l.Screens)-1 {
		return l.ActiveID, fmt.Errorf("already at last screen")
	}
	l.ActiveID++
	l.CursorX = 0
	return l.ActiveID, nil
}

// SwitchPrev moves focus to the previous screen (hotkey: Ctrl+Alt+←).
func (l *Layout) SwitchPrev() (uint8, error) {
	if l.ActiveID == 0 {
		return l.ActiveID, fmt.Errorf("already at first screen")
	}
	l.ActiveID--
	l.CursorX = 0
	return l.ActiveID, nil
}

// RevertSwitch undoes a switch when the target screen has no connected client.
func (l *Layout) RevertSwitch(prevID uint8, movedRight bool) {
	l.ActiveID = prevID
	l.CursorX = 0
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
