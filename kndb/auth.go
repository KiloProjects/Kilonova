package kndb

import (
	"fmt"

	"github.com/KiloProjects/Kilonova/common"
	"golang.org/x/crypto/bcrypt"
)

// RegisterUser registers a user in the DB and returns a user instance to use later
func (d *DB) RegisterUser(email, username, password string) (*common.User, error) {
	var user common.User
	user.Name = username
	user.Email = email
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		return nil, err
	}
	fmt.Println(string(hashed))
	user.Password = string(hashed)
	d.DB.Create(&user)
	if user.ID == 1 {
		d.MakeAdmin(1)
	}
	return &user, nil
}
