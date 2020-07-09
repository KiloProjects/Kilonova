// Package common contains stuff that can be used by all 4 components of the project (grader, API server, data manager and web UI)
package common

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	"github.com/gorilla/securecookie"
	"github.com/jinzhu/gorm"
)

// DataDir should be where most of the data is stored
const DataDir = "/data"

// Config is the main information for the platform
type Config struct {
	Debug bool `json:"debugMode"`
}

// Session represents the data storred in a session cookie
type Session struct {
	UserID uint `json:"userID"`
}

// Updater is an interface for a DB updater made by the boxManager (like updating the status of a task)
type Updater interface {
	Update(*gorm.DB) error
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
