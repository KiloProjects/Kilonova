package common

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
)

// this file stores stuff to both the server and web parts

// KNContextType is the string type for all context values
type KNContextType string

// Cookies is the securecookie instance that should be called by everyone
var Cookies *securecookie.SecureCookie

const (
	// UserKey is the key to be used for adding user objects to context
	UserKey = KNContextType("user")
	// PbID is the key to be used for adding problem IDs to context
	PbID = KNContextType("pbID")
	// ProblemKey is the key to be used for adding problems to context
	ProblemKey = KNContextType("problem")
	// TaskID  is the key to be used for adding task IDs to context
	TaskID = KNContextType("taskID")
	// TaskKey is the key to be used for adding tasks to context
	TaskKey = KNContextType("task")
)

// RetData should be the way data is sent between the API and the Client
type RetData struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// UserFromContext returns the user from context
func UserFromContext(r *http.Request) User {
	switch v := r.Context().Value(UserKey).(type) {
	case User:
		return v
	default:
		return User{}
	}
}

// ProblemFromContext returns the problem from context
func ProblemFromContext(r *http.Request) Problem {
	switch v := r.Context().Value(ProblemKey).(type) {
	case Problem:
		return v
	default:
		return Problem{}
	}
}

// TaskFromContext returns the task from context
func TaskFromContext(r *http.Request) Task {
	switch v := r.Context().Value(TaskKey).(type) {
	case Task:
		return v
	default:
		return Task{}
	}
}

// GetSession reads and returns the data from the session cookie
func GetSession(r *http.Request) *Session {
	authToken := GetAuthToken(r)
	if authToken != "" { // use Auth tokens by default
		var ret Session
		Cookies.Decode("kn-sessionid", authToken, &ret)
		return &ret
	}
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		return nil
	}
	if cookie.Value == "" {
		return nil
	}
	var ret Session
	Cookies.Decode(cookie.Name, cookie.Value, &ret)
	return &ret
}

// GetAuthToken returns the authentication token from a request
func GetAuthToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

// IsAuthed reads the session and says if the requester is authenticated
func IsAuthed(r *http.Request) bool {
	session := GetSession(r)
	if session == nil {
		return false
	}
	return session.UserID != 0
}

// IsAdmin reads the session and says if the requester is an admin
func IsAdmin(r *http.Request) bool {
	session := GetSession(r)
	if session == nil {
		return false
	}
	return session.IsAdmin
}

// SetSession sets the data to the session cookie
func SetSession(w http.ResponseWriter, sess Session) (string, error) {
	encoded, err := Cookies.Encode("kn-sessionid", sess)
	if err != nil {
		return "", err
	}
	cookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    encoded,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteDefaultMode,
	}
	http.SetCookie(w, cookie)
	return encoded, nil
}

// RemoveSessionCookie clears the session cookie, effectively revoking it. When setting MaxAge to 0, the browser will also clear it out
func RemoveSessionCookie(w http.ResponseWriter) {
	emptyCookie := &http.Cookie{
		Name:    "kn-sessionid",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, emptyCookie)
}

func init() {
	// generate secure cookie
	var secure []byte
	if _, err := os.Stat(path.Join(DataDir, "secretKey")); os.IsNotExist(err) {
		secure = securecookie.GenerateRandomKey(32)
		ioutil.WriteFile(path.Join(DataDir, "secretKey"), secure, 0777)
	} else {
		secure, err = ioutil.ReadFile(path.Join(DataDir, "secretKey"))
		secure = bytes.TrimSpace(secure)
	}
	if len(secure) != 32 {
		panic("Invalid secure key length")
	}
	Cookies = securecookie.New(secure, nil).SetSerializer(securecookie.JSONEncoder{})
}
