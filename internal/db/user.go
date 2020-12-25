package db

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
)

type User struct {
	ctx   context.Context
	db    *DB
	Empty bool `json:"-"`

	ID             int64     `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	Name           string    `json:"name"`
	Admin          bool      `json:"admin"`
	Proposer       bool      `json:"proposer"`
	Email          string    `json:"email"`
	Password       string    `json:"-"`
	Bio            string    `json:"bio"`
	DefaultVisible bool      `json:"default_visible"`
}

func (u *User) SetAdmin(admin bool) error {
	if err := u.db.raw.SetAdmin(u.ctx, rawdb.SetAdminParams{ID: u.ID, Admin: admin}); err != nil {
		return err
	}

	u.Admin = admin
	return nil
}

func (u *User) SetProposer(proposer bool) error {
	if err := u.db.raw.SetProposer(u.ctx, rawdb.SetProposerParams{ID: u.ID, Proposer: proposer}); err != nil {
		return err
	}

	u.Proposer = proposer
	return nil
}

func (u *User) SetEmail(email string) error {
	if err := u.db.raw.SetEmail(u.ctx, rawdb.SetEmailParams{ID: u.ID, Email: email}); err != nil {
		return err
	}

	u.Email = email
	return nil
}

func (u *User) SetBio(bio string) error {
	if err := u.db.raw.SetBio(u.ctx, rawdb.SetBioParams{ID: u.ID, Bio: bio}); err != nil {
		return err
	}

	u.Bio = bio
	return nil
}

func (u *User) SetDefaultVisibility(visible bool) error {
	if err := u.db.raw.SetDefaultVisibility(u.ctx, rawdb.SetDefaultVisibilityParams{ID: u.ID, DefaultVisible: visible}); err != nil {
		return err
	}

	u.DefaultVisible = visible
	return nil
}

func (u *User) SetPasswordHash(hash string) error {
	if err := u.db.raw.SetPassword(u.ctx, rawdb.SetPasswordParams{ID: u.ID, Password: hash}); err != nil {
		return err
	}

	u.Password = hash
	return nil
}

func (u *User) MaxScore(pbID int64) int {
	return u.db.MaxScore(u.ctx, u.ID, pbID)
}

func (u *User) ProblemSubs(pbID int64) ([]*Submission, error) {
	subs, err := u.db.raw.UserProblemSubmissions(u.ctx, rawdb.UserProblemSubmissionsParams{UserID: u.ID, ProblemID: pbID})
	if err != nil {
		return nil, err
	}

	var submissions []*Submission
	for _, sub := range subs {
		submissions = append(submissions, u.db.subFromRaw(u.ctx, sub))
	}
	return submissions, nil
}

func (u *User) SolvedProblemIDs() ([]int64, error) {
	return u.db.raw.SolvedProblems(u.ctx, u.ID)
}

func (u *User) SolvedProblems() ([]*Problem, error) {
	ids, err := u.SolvedProblemIDs()
	if err != nil {
		return nil, err
	}
	var problems []*Problem
	for _, id := range ids {
		pb, err := u.db.Problem(u.ctx, id)
		if err != nil {
			return problems, err
		}

		problems = append(problems, pb)
	}
	return problems, nil
}

func (db *DB) UserExists(ctx context.Context, username, email string) bool {
	cnt, err := db.raw.CountUsers(ctx, rawdb.CountUsersParams{Username: username, Email: email})
	if err != nil {
		log.Println(err)
		return true
	}

	return cnt > 0
}

func (db *DB) User(ctx context.Context, id int64) (*User, error) {
	user, err := db.raw.User(ctx, id)
	if err != nil {
		return nil, err
	}
	return db.userFromRaw(ctx, user), nil
}

func (db *DB) UserByName(ctx context.Context, name string) (*User, error) {
	user, err := db.raw.UserByName(ctx, strings.ToLower(name))
	if err != nil {
		return nil, err
	}
	return db.userFromRaw(ctx, user), nil
}

func (db *DB) Users(ctx context.Context) ([]*User, error) {
	users, err := db.raw.Users(ctx)
	if err != nil {
		return nil, err
	}
	return db.usersFromRaw(ctx, users), nil
}

func (db *DB) Admins(ctx context.Context) ([]*User, error) {
	users, err := db.raw.Admins(ctx)
	if err != nil {
		return nil, err
	}
	return db.usersFromRaw(ctx, users), nil
}

func (db *DB) Proposers(ctx context.Context) ([]*User, error) {
	users, err := db.raw.Proposers(ctx)
	if err != nil {
		return nil, err
	}
	return db.usersFromRaw(ctx, users), nil
}

type Top100Row struct {
	User           *User `json:"user"`
	NumberProblems int   `json:"number_problems"`
}

// Top100 returns top 100 users with their number of problems
func (db *DB) Top100(ctx context.Context) ([]Top100Row, error) {
	rows, err := db.raw.Top100(ctx)
	if err != nil {
		return nil, err
	}

	var retRows []Top100Row
	for _, row := range rows {
		user := &User{
			ctx: ctx,
			db:  db,

			ID:             row.ID,
			CreatedAt:      row.CreatedAt.UTC(),
			Name:           row.Name,
			Admin:          row.Admin,
			Proposer:       row.Proposer,
			Email:          row.Email,
			Bio:            row.Bio,
			DefaultVisible: row.DefaultVisible,
		}
		retRows = append(retRows, Top100Row{User: user, NumberProblems: int(row.NumberProblems)})
	}

	return retRows, nil
}

/////////////////////////////////////////////

func (db *DB) CreateUser(ctx context.Context, username, email, hash string) (*User, error) {
	user, err := db.raw.CreateUser(ctx, rawdb.CreateUserParams{Name: username, Email: email, Password: hash})
	if err != nil {
		return nil, err
	}

	return db.userFromRaw(ctx, user), nil
}

/////////////////////////////////////////////

func (db *DB) userFromRaw(ctx context.Context, u rawdb.User) *User {
	return &User{
		ctx: ctx,
		db:  db,

		ID:             u.ID,
		CreatedAt:      u.CreatedAt.UTC(),
		Name:           u.Name,
		Admin:          u.Admin,
		Proposer:       u.Proposer,
		Email:          u.Email,
		Password:       u.Password,
		Bio:            u.Bio,
		DefaultVisible: u.DefaultVisible,
	}
}

func (db *DB) usersFromRaw(ctx context.Context, usrs []rawdb.User) []*User {
	var users []*User
	for _, user := range usrs {
		users = append(users, db.userFromRaw(ctx, user))
	}

	return users
}
