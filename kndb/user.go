package kndb

import (
	"fmt"

	"github.com/KiloProjects/Kilonova/common"
)

// GetUserByID returns a user from the ID
// If the user does not exist (or something happened), it provides an error
func (d *DB) GetUserByID(id uint) (*common.User, error) {
	var user common.User
	if err := d.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	if user.ID == 0 {
		// this should never happen, but just in case it happens, log it so i can fix it if it ever happens
		fmt.Println("User ID is 0, huh?")
	}
	return &user, nil
}

// GetUserByName returns a user from the username (note that this is case-insensitive)
// If the user does not exist (or something happened), it provides an error
func (d *DB) GetUserByName(name string) (*common.User, error) {
	var user common.User
	if err := d.DB.First(&user, "lower(name) = lower(?)", name).Error; err != nil {
		return nil, err
	}
	if user.ID == 0 {
		// this should never happen, but just in case it happens, log it so i can fix it
		fmt.Println("User ID is 0, huh?")
	}
	return &user, nil
}

// GetUserByEmail returns a user from the email (note that this is case-insensitive)
// If the user does not exist (or something happened), it provides an error
func (d *DB) GetUserByEmail(email string) (*common.User, error) {
	var user common.User
	if err := d.DB.First(&user, "lower(email) = lower(?)", email).Error; err != nil {
		return nil, err
	}
	if user.ID == 0 {
		// this should never happen, but just in case it happens, log it so i can fix it
		fmt.Println("User ID is 0, huh?")
	}
	return &user, nil
}

// UserExists returns a bool indicating if the user with a specified email or username exists
// Note that if an argument is empty (ie, it's equal to ""), it's ignored
func (d *DB) UserExists(email string, username string) bool {
	var cnt int64
	if email != "" {
		if err := d.DB.Model(&common.User{}).Where("lower(email) = lower(?)", email).Count(&cnt).Error; err != nil {
			fmt.Println("Error counting in DB:", err)
			return false
		}
		if cnt > 0 {
			return true
		}
	}
	if username != "" {
		if err := d.DB.Model(&common.User{}).Where("lower(name) = lower(?)", username).Count(&cnt).Error; err != nil {
			fmt.Println("Error counting in DB:", err)
			return false
		}
		if cnt > 0 {
			return true
		}
	}
	return false
}

// GetAllUsers returns a slice of all users
// TODO: Should add pagination later
func (d *DB) GetAllUsers() ([]common.User, error) {
	var users []common.User
	if err := d.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (d *DB) GetAllAdmins() ([]common.User, error) {
	var users []common.User
	if err := d.DB.Where("admin = ?", true).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (d *DB) GetAllProposers() ([]common.User, error) {
	var users []common.User
	if err := d.DB.Where("proposer = ? or admin = ?", true, true).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (d *DB) SetAdmin(id uint, on bool) error {
	return d.DB.Model(&common.User{}).Where("id = ?", id).Update("admin", on).Error
}

func (d *DB) SetProposer(id uint, on bool) error {
	return d.DB.Model(&common.User{}).Where("id = ?", id).Update("proposer", on).Error
}

// SetEmail sets the email of a user with the set ID
func (d *DB) SetEmail(id uint, email string) error {
	var user common.User
	user.ID = id
	return d.DB.Model(&user).Update("email", email).Error
}
