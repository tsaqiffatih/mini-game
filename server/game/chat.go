package game

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	maxChatMessages     = 50
	maxChatMessageChars = 300
)

var chatMessageSeq uint64

var (
	ErrChatMessageEmpty   = errors.New("chat message is empty")
	ErrChatMessageTooLong = errors.New("chat message is too long")
)

type ChatMessage struct {
	ID         string
	RoomID     string
	PlayerID   string
	PlayerMark string
	Message    string
	CreatedAt  time.Time
}

func newChatMessageID(now time.Time) string {
	seq := atomic.AddUint64(&chatMessageSeq, 1)
	return fmt.Sprintf("msg_%d_%d", now.UnixNano(), seq)
}

func normalizeChatMessage(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", ErrChatMessageEmpty
	}
	if utf8.RuneCountInString(message) > maxChatMessageChars {
		return "", ErrChatMessageTooLong
	}
	return message, nil
}
