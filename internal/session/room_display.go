package session

import (
	"fmt"

	"github.com/luojam/ssh-chat/internal/chat"
	"github.com/luojam/ssh-chat/internal/tui"
)

const (
	displayLocalAuthor   = "you"
	displaySystemAuthor  = "[system]"
	displayUnknownAuthor = "unknown"
)

func (m model) displayMessage(event chat.Event) (tui.MessageReceived, bool) {
	switch event.Kind {
	case chat.MessagePosted:
		author := displayMemberName(event.Message.Author.Name)
		role := tui.MessageAuthorNormal
		if event.Message.Author.ID == m.member.ID {
			author = displayLocalAuthor
			role = tui.MessageAuthorLocal
		}
		return tui.MessageReceived{
			Author: author,
			Body:   event.Message.Body,
			Role:   role,
		}, true
	case chat.MemberJoined:
		return tui.MessageReceived{
			Author: displaySystemAuthor,
			Body:   fmt.Sprintf("%s joined", displayMemberName(event.Member.Name)),
			Role:   tui.MessageAuthorSystem,
		}, true
	case chat.MemberLeft:
		return tui.MessageReceived{
			Author: displaySystemAuthor,
			Body:   fmt.Sprintf("%s left", displayMemberName(event.Member.Name)),
			Role:   tui.MessageAuthorSystem,
		}, true
	default:
		return tui.MessageReceived{}, false
	}
}

func displayMemberName(name string) string {
	if name == "" {
		return displayUnknownAuthor
	}
	return name
}
