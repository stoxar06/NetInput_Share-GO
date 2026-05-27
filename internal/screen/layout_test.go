package screen

import "testing"

func fourScreenLayout() *Layout {
	return NewLayout([]Screen{
		{ID: 0, Name: "Main", Width: 1920, Height: 1080},
		{ID: 1, Name: "L2", Width: 1920, Height: 1080},
		{ID: 2, Name: "L3", Width: 1920, Height: 1080},
		{ID: 3, Name: "L4", Width: 1920, Height: 1080},
	}, "horizontal")
}

func TestUpdateCursor_noSwitch(t *testing.T) {
	l := fourScreenLayout()
	newID, switched := l.UpdateCursor(100, 50)
	if switched {
		t.Error("expected no switch")
	}
	if newID != 0 {
		t.Errorf("expected screenID 0, got %d", newID)
	}
}

func TestUpdateCursor_switchRight(t *testing.T) {
	l := fourScreenLayout()
	// Start at centre (960). Need to push past 1920.
	newID, switched := l.UpdateCursor(1920, 0)
	if !switched {
		t.Error("expected switch when cursor crosses right edge")
	}
	if newID != 1 {
		t.Errorf("expected screenID 1, got %d", newID)
	}
	if l.CursorX != EntryX {
		t.Errorf("expected CursorX=%d on entry, got %d", EntryX, l.CursorX)
	}
}

func TestUpdateCursor_switchRightThenBack(t *testing.T) {
	l := fourScreenLayout()
	l.UpdateCursor(1920, 0)        // switch to screen 1; CursorX = EntryX
	newID, switched := l.UpdateCursor(-1920, 0) // cross left edge
	if !switched {
		t.Error("expected switch back to screen 0")
	}
	if newID != 0 {
		t.Errorf("expected screenID 0, got %d", newID)
	}
}

func TestUpdateCursor_clampAtLeftEdge(t *testing.T) {
	l := fourScreenLayout()
	newID, switched := l.UpdateCursor(-9999, 0)
	if switched {
		t.Error("expected no switch at leftmost screen")
	}
	if newID != 0 {
		t.Error("expected to stay on screen 0")
	}
	if l.CursorX != 0 {
		t.Errorf("expected CursorX clamped to 0, got %d", l.CursorX)
	}
}

func TestUpdateCursor_clampAtRightEdge(t *testing.T) {
	l := fourScreenLayout()
	// Move to last screen.
	l.UpdateCursor(1920, 0)
	l.UpdateCursor(1920, 0)
	l.UpdateCursor(1920, 0) // now on screen 3
	newID, switched := l.UpdateCursor(9999, 0)
	if switched {
		t.Error("expected no switch past last screen")
	}
	if newID != 3 {
		t.Errorf("expected screenID 3, got %d", newID)
	}
	if l.CursorX != 1919 {
		t.Errorf("expected CursorX clamped to 1919, got %d", l.CursorX)
	}
}

func TestUpdateCursor_clampY(t *testing.T) {
	l := fourScreenLayout()
	l.UpdateCursor(0, -999)
	if l.CursorY != 0 {
		t.Errorf("expected CursorY clamped to 0, got %d", l.CursorY)
	}
	l.UpdateCursor(0, 9999)
	if l.CursorY != 1079 {
		t.Errorf("expected CursorY clamped to 1079, got %d", l.CursorY)
	}
}

func TestSwitchNext(t *testing.T) {
	l := fourScreenLayout()
	id, err := l.SwitchNext()
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Errorf("expected 1, got %d", id)
	}
}

func TestSwitchPrev_atFirst(t *testing.T) {
	l := fourScreenLayout()
	_, err := l.SwitchPrev()
	if err == nil {
		t.Error("expected error when switching prev from first screen")
	}
}

func TestSwitchNext_atLast(t *testing.T) {
	l := fourScreenLayout()
	l.ActiveID = 3
	_, err := l.SwitchNext()
	if err == nil {
		t.Error("expected error when switching next from last screen")
	}
}

func TestSwitchNextPrev_roundtrip(t *testing.T) {
	l := fourScreenLayout()
	for i := 0; i < 3; i++ {
		l.SwitchNext()
	}
	if l.ActiveID != 3 {
		t.Errorf("expected activeID 3 after 3 nexts, got %d", l.ActiveID)
	}
	for i := 0; i < 3; i++ {
		l.SwitchPrev()
	}
	if l.ActiveID != 0 {
		t.Errorf("expected activeID 0 after 3 prevs, got %d", l.ActiveID)
	}
}
