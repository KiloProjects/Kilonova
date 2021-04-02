package logic

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
)

// GetSession reads and returns the data from the session cookie
func (kn *Kilonova) GetRSession(r *http.Request) int {
	authToken := getAuthHeader(r)
	if authToken != "" { // use Auth tokens by default
		id, err := kn.GetSession(authToken)
		if err == nil {
			return id
		}
	}
	return -1
}

func (kn *Kilonova) CreateSession(id int) (string, error) {
	return kn.Sess.CreateSession(context.Background(), id)
}

func (kn *Kilonova) GetSession(id string) (int, error) {
	return kn.Sess.GetSession(context.Background(), id)
}

// RemoveSessionCookie clears the session cookie
func (kn *Kilonova) RemoveSessionCookie(w http.ResponseWriter, r *http.Request) {
	emptyCookie := &http.Cookie{
		Name:    "kn-sessionid",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, emptyCookie)

	c, err := r.Cookie("kn-sessionid")
	if err != nil {
		return
	}
	kn.Sess.RemoveSession(context.Background(), c.Value)
}

func (kn *Kilonova) GenHash(password string) (string, error) {
	return kilonova.HashPassword(password)
}

func (kn *Kilonova) AddUser(ctx context.Context, username, email, password string) (*kilonova.User, error) {
	hash, err := kn.GenHash(password)
	if err != nil {
		return nil, err
	}

	var user kilonova.User
	user.Name = username
	user.Email = email
	user.Password = hash

	err = kn.userv.CreateUser(ctx, &user)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if user.ID == 1 {
		var True = true
		if err := kn.userv.UpdateUser(ctx, user.ID, kilonova.UserUpdate{Admin: &True, Proposer: &True}); err != nil {
			log.Println(err)
			return &user, err
		}
	}

	return &user, nil
}

func getAuthHeader(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "guest" {
		header = ""
	}
	return header
}
