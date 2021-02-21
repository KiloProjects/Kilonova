package psql

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/matryer/is"
)

var (
	userv  kilonova.UserService
	tserv  kilonova.TestService
	sserv  kilonova.SubmissionService
	stserv kilonova.SubTestService
	pserv  kilonova.ProblemService

	testUser = flag.Bool("testUser", false, "")
	testTest = flag.Bool("testTest", false, "")
	testSubs = flag.Bool("testSubs", false, "")
)

func bPtr(b bool) *bool     { return &b }
func sPtr(s string) *string { return &s }
func iPtr(i int) *int       { return &i }

var (
	ctx = context.Background()
)

func TestUserByID(t *testing.T) {
	if !*testUser {
		t.SkipNow()
	}
	is := is.New(t)

	user, err := userv.UserByID(ctx, 1)
	is.NoErr(err)
	t.Logf("%#v\n", user)
}

func TestUserFilters(t *testing.T) {
	if !*testUser {
		t.SkipNow()
	}
	is := is.New(t)

	filter := kilonova.UserFilter{
		Proposer: bPtr(true),
		Admin:    bPtr(false),
	}

	users, err := userv.Users(ctx, filter)
	is.NoErr(err)
	for _, user := range users {
		t.Logf("%#v\n", user)
	}
}

func TestUserCounting(t *testing.T) {
	if !*testUser {
		t.SkipNow()
	}
	is := is.New(t)

	filter := kilonova.UserFilter{
		//Proposer: &True,
		//Admin:    &True,
	}
	cnt, err := userv.CountUsers(ctx, filter)
	is.NoErr(err)
	t.Logf("%d\n", cnt)
}

func TestUserExists(t *testing.T) {
	if !*testUser {
		t.SkipNow()
	}
	is := is.New(t)

	exists, err := userv.UserExists(ctx, "alexvasiluta", "a@a.a")
	is.NoErr(err)
	t.Logf("%t\n", exists)
}

func TestUserCreating(t *testing.T) {
	if !*testUser || testing.Short() {
		t.SkipNow()
	}
	is := is.New(t)
	user := kilonova.User{
		Name:     "TestingStuff",
		Password: "asdfghj",
		Email:    "kilonova@kilonova.kilonova",
	}
	// When running on an active DB, this will auto increment the user ID. I do not plan on fixing this problem.
	is.NoErr(userv.CreateUser(ctx, &user))
	is.NoErr(userv.DeleteUser(ctx, user.ID))
}

func TestMain(m *testing.M) {
	flag.Parse()
	fmt.Println(*testUser, *testTest, *testSubs)
	if err := config.Load("../config.toml"); err != nil {
		panic(err)
	}
	db, err := New(config.Database.String())
	if err != nil {
		panic(err)
	}
	userv = db.UserService()
	tserv = db.TestService()
	sserv = db.SubmissionService()
	stserv = db.SubTestService()
	pserv = db.ProblemService()
	code := m.Run()
	if code != 0 {
		os.Exit(code)
	}
	if err := db.Close(); err != nil {
		panic(err)
	}
}
