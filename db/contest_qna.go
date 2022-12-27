package db

import (
	"context"

	"github.com/KiloProjects/kilonova"
)

// Contest Questions/Answers and Announcements

func (s *DB) CreateContestQuestion(ctx context.Context, contestID, authorID int, text string) error {
	panic("TODO")
}

func (s *DB) ContestQuestions(ctx context.Context, contestID int) ([]*kilonova.ContestQuestion, error) {
	panic("TODO")
}

func (s *DB) AnswerContestQuestion(ctx context.Context, questionID int) {
	panic("TODO")
}

func (s *DB) CreateContestAnnouncement(ctx context.Context, contestID, authorID int, text string) error {
	panic("TODO")
}

// TODO: Update annonucement, delete question/announcement
