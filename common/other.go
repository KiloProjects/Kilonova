// Package common contains stuff that can be used by all 4 components of the project (grader, API server, data manager and web UI)
package common

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/gorilla/securecookie"
)

// DataDir should be where most of the data is stored
var DataDir = "/data"

const Version = "v0.2.0"

// Config is the main information for the platform
type Config struct {
	Debug bool `json:"debugMode"`
}

// Session represents the data storred in a session cookie
type Session struct {
	UserID uint `json:"userID"`
}

// SetDataDir sets DataDir to the specified path
func SetDataDir(d string) {
	DataDir = d
}

// Initialize must be called after setting the DataDir, but before everything else
func Initialize() {
	if err := os.MkdirAll(DataDir, 0775); err != nil {
		panic(err)
	}

	// generate secure cookie
	var secure []byte
	if _, err := os.Stat(path.Join(DataDir, "secretKey")); os.IsNotExist(err) {
		secure = securecookie.GenerateRandomKey(32)
		ioutil.WriteFile(path.Join(DataDir, "secretKey"), secure, 0777)
	} else {
		secure, err = ioutil.ReadFile(path.Join(DataDir, "secretKey"))
		if err != nil {
			panic(fmt.Sprintln("Could not read the secret key:", err))
		}
		secure = bytes.TrimSpace(secure)
	}
	if len(secure) != 32 {
		panic("Invalid secure key length")
	}
	Cookies = securecookie.New(secure, nil).SetSerializer(securecookie.JSONEncoder{})
}
