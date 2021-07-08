package kilonova

import (
	"context"
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

func SolvedProblems(ctx context.Context, uid int, db DB) ([]*Problem, error) {
	ids, err := db.SolvedProblems(ctx, uid)
	if err != nil {
		return nil, err
	}
	var pbs = make([]*Problem, 0, len(ids))
	for _, id := range ids {
		pb, err := db.Problem(ctx, id)
		if err != nil {
			log.Printf("Couldn't get solved problem %d: %s\n", id, err)
		} else {
			pbs = append(pbs, pb)
		}
	}
	return pbs, nil
}

func VisibleProblems(ctx context.Context, user *User, db DB) ([]*Problem, error) {
	var uid int
	if user != nil {
		uid = user.ID
		if user.Admin {
			uid = -1
		}
	}
	return db.Problems(ctx, ProblemFilter{LookingUserID: &uid})
}

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
