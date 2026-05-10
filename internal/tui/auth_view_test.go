package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestAuthBoxHeightDoesNotChangeBetweenLoginAndSignup(t *testing.T) {
	m := NewAuth(Config{Width: 80, Height: 30}).(authModel)
	initial := authLayoutFor(m.screen.width, m.screen.height, m.styles, 1)
	width := initial.content.width

	m.mode = AuthModeLogin
	loginLayout := authLayoutFor(m.screen.width, m.screen.height, m.styles, m.desiredContentHeight(width))
	loginBody := m.renderAuthBody(width)

	m.mode = AuthModeSignup
	signupLayout := authLayoutFor(m.screen.width, m.screen.height, m.styles, m.desiredContentHeight(width))
	signupBody := m.renderAuthBody(width)

	if loginLayout.box.height != signupLayout.box.height {
		t.Fatalf("login box height = %d, signup box height = %d; want same", loginLayout.box.height, signupLayout.box.height)
	}

	loginBottomPadding := loginLayout.content.height - authVerticalPaddingRows - lipgloss.Height(loginBody)
	if loginBottomPadding != 4 {
		t.Fatalf("login bottom padding = %d, want 4", loginBottomPadding)
	}

	signupBottomPadding := signupLayout.content.height - authVerticalPaddingRows - lipgloss.Height(signupBody)
	if signupBottomPadding != authVerticalPaddingRows {
		t.Fatalf("signup bottom padding = %d, want %d", signupBottomPadding, authVerticalPaddingRows)
	}
}

func TestAuthBoxHeightShrinksWithViewport(t *testing.T) {
	m := NewAuth(Config{Width: 80, Height: 8}).(authModel)
	initial := authLayoutFor(m.screen.width, m.screen.height, m.styles, 1)
	layout := authLayoutFor(m.screen.width, m.screen.height, m.styles, m.desiredContentHeight(initial.content.width))

	maxHeight := m.screen.height - authFramePaddingY*2
	if layout.box.height != maxHeight {
		t.Fatalf("box height = %d, want viewport-capped height %d", layout.box.height, maxHeight)
	}
}
