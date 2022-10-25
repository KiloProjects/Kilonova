package kilonova

import (
	"log"
	"strconv"
	"strings"
)

type Mailer interface {
	SendEmail(msg *MailerMessage) error
}

type MailerMessage struct {
	To      string
	Subject string
	ReplyTo string

	PlainContent string
	HTMLContent  string
}

type MarkdownRenderer interface {
	Render(src []byte) ([]byte, error)
}

///// Utility functions that require multiple services

func SerializeIntList(ids []int) string {
	if ids == nil {
		return ""
	}
	var b strings.Builder
	for i, id := range ids {
		b.WriteString(strconv.Itoa(id))
		if i != len(ids)-1 {
			b.WriteRune(',')
		}
	}
	return b.String()
}

func DeserializeIntList(ids string) []int {
	ll := strings.Split(ids, ",")
	l := make([]int, 0, len(ll))
	for _, str := range ll {
		val, err := strconv.Atoi(str)
		if err != nil {
			log.Println("WARNING: An invalid integer slipped in an int list")
			continue
		}
		l = append(l, val)
	}
	return l
}
