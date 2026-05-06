package tui

import "testing"

func TestSafeFrameSizeSanitizesDimensions(t *testing.T) {
	frame := safeFrameSize(0, -2)

	if frame.width != 1 || frame.height != 1 {
		t.Fatalf("safeFrameSize(0, -2) = %+v, want width=1 height=1", frame)
	}
}
