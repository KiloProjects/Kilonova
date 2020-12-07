package logic

import (
	"context"
	"log"

	"github.com/KiloProjects/Kilonova/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// TODO: Move session handling here

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
