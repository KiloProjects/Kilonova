package kilonova

import (
	"context"
	"database/sql"
	"errors"
	"io"
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

type Sessioner interface {
	CreateSession(ctx context.Context, uid int) (string, error)
	GetSession(ctx context.Context, sess string) (int, error)
	RemoveSession(ctx context.Context, sess string) error
}

type Verificationer interface {
	CreateVerification(ctx context.Context, id int) (string, error)
	GetVerification(ctx context.Context, verif string) (int, error)
	RemoveVerification(ctx context.Context, verif string) error
}

// TypeServicer is an interface for a provider for UserService, ProblemService, TestService, SubmissionService and SubTestService
type TypeServicer interface {
	UserService() UserService
	ProblemService() ProblemService
	TestService() TestService
	SubmissionService() SubmissionService
	SubTestService() SubTestService
	ProblemListService() ProblemListService
	SubTaskService() SubTaskService
	SessionService() Sessioner
	VerificationService() Verificationer
	io.Closer
}

///// Utility functions that require multiple services

func SolvedProblems(ctx context.Context, uid int, sserv SubmissionService, pserv ProblemService) ([]*Problem, error) {
	ids, err := sserv.SolvedProblems(ctx, uid)
	if err != nil {
		return nil, err
	}
	var pbs = make([]*Problem, 0, len(ids))
	for _, id := range ids {
		pb, err := pserv.ProblemByID(ctx, id)
		if err != nil {
			log.Printf("Couldn't get solved problem %d: %s\n", id, err)
		} else {
			pbs = append(pbs, pb)
		}
	}
	return pbs, nil
}

func VisibleProblems(ctx context.Context, user *User, pserv ProblemService) (pbs []*Problem, err error) {
	if user != nil && user.Admin {
		pbs, err = pserv.Problems(ctx, ProblemFilter{})
	} else {
		var uid int
		if user != nil {
			uid = user.ID
		}
		pbs, err = pserv.Problems(ctx, ProblemFilter{LookingUserID: &uid})
	}
	if errors.Is(err, sql.ErrNoRows) {
		return []*Problem{}, nil
	}
	return
}

func InsertArchive(ctx context.Context, owner *User, pbs []*FullProblem, pserv ProblemService, tserv TestService, stkserv SubTaskService, store GraderStore) error {
	panic("TODO")
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
