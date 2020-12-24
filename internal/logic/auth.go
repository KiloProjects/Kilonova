package logic

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/KiloProjects/Kilonova/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// GetSession reads and returns the data from the session cookie
func (kn *Kilonova) GetRSession(r *http.Request) int64 {
	authToken := getAuthHeader(r)
	if authToken != "" { // use Auth tokens by default
		id, err := kn.GetSession(authToken)
		if err == nil {
			return id
		}
	}
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		return -1
	}
	if cookie.Value == "" {
		return -1
	}
	id, err := kn.GetSession(cookie.Value)
	if err != nil {
		return -1
	}
	return id
}

// SetSession sets the data to the session cookie
func (kn *Kilonova) SetRSession(w http.ResponseWriter, id int64) (string, error) {
	sid, err := kn.CreateSession(id)
	if err != nil {
		log.Println(err)
		return "", err
	}
	cookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    sid,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteDefaultMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	}
	http.SetCookie(w, cookie)
	return sid, nil
}

func (kn *Kilonova) CreateSession(id int64) (string, error) {
	return kn.RClient.CreateSession(context.Background(), id)
}

func (kn *Kilonova) GetSession(id string) (int64, error) {
	return kn.RClient.GetSession(context.Background(), id)
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
	kn.RClient.RemoveSession(context.Background(), c.Value)
}

func (kn *Kilonova) GenHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), err
}

func (kn *Kilonova) AddUser(ctx context.Context, username, email, password string) (*db.User, error) {
	hash, err := kn.GenHash(password)
	if err != nil {
		return nil, err
	}

	user, err := kn.DB.CreateUser(ctx, username, email, hash)

	if user.ID == 1 {
		if err := user.SetAdmin(true); err != nil {
			log.Println(err)
			return user, err
		}
		if err := user.SetProposer(true); err != nil {
			log.Println(err)
			return user, err
		}
	}

	return user, nil
}

func (kn *Kilonova) ValidCreds(ctx context.Context, username, password string) (*db.User, error) {
	return nil, nil
}

func getAuthHeader(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}
