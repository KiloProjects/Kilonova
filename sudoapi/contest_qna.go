package sudoapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
)

func (s *BaseAPI) CreateContestQuestion(ctx context.Context, contest *kilonova.Contest, authorID int, text string) (int, error) {
	if contest.QuestionCooldown > 0 {
		question, err := s.db.ContestQuestions(ctx, db.QuestionFilter{ContestID: &contest.ID, AuthorID: &authorID})
		if err != nil {
			return -1, fmt.Errorf("could not check for question cooldown: %w", err)
		}
		if len(question) > 0 {
			if d := contest.QuestionCooldown - time.Since(question[0].AskedAt); d > 0 {
				return -1, Statusf(http.StatusTooManyRequests, "You are going too fast! Please wait %d more second(s) before asking another question.", int(d.Seconds())+1)
			}
		}
	}

	id, err := s.db.CreateContestQuestion(ctx, contest.ID, authorID, text)
	if err != nil {
		return -1, fmt.Errorf("couldn't ask question: %w", err)
	}
	return id, nil
}

func (s *BaseAPI) CreateContestAnnouncement(ctx context.Context, contestID int, text string) (int, error) {
	id, err := s.db.CreateContestAnnouncement(ctx, contestID, text)
	if err != nil {
		return -1, fmt.Errorf("couldn't create announcement: %w", err)
	}
	return id, nil
}

func (s *BaseAPI) ContestAnnouncements(ctx context.Context, contestID int) ([]*kilonova.ContestAnnouncement, error) {
	announcements, err := s.db.ContestAnnouncements(ctx, contestID)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch announcements: %w", err)
	}
	return announcements, nil
}

func (s *BaseAPI) ContestAnnouncement(ctx context.Context, id int) (*kilonova.ContestAnnouncement, error) {
	announcement, err := s.db.ContestAnnouncement(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch announcement: %w", err)
	}
	if announcement == nil {
		return nil, fmt.Errorf("couldn't fetch announcement: %w", ErrNotFound)
	}
	return announcement, nil
}

func (s *BaseAPI) ContestQuestions(ctx context.Context, contestID int) ([]*kilonova.ContestQuestion, error) {
	questions, err := s.db.ContestQuestions(ctx, db.QuestionFilter{ContestID: &contestID})
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch questions: %w", err)
	}
	return questions, nil
}

func (s *BaseAPI) ContestQuestion(ctx context.Context, id int) (*kilonova.ContestQuestion, error) {
	question, err := s.db.ContestQuestion(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch question: %w", err)
	}
	if question == nil {
		return nil, fmt.Errorf("couldn't fetch question: %w", ErrNotFound)
	}
	return question, nil
}

func (s *BaseAPI) ContestUserQuestions(ctx context.Context, contestID, userID int) ([]*kilonova.ContestQuestion, error) {
	questions, err := s.db.ContestQuestions(ctx, db.QuestionFilter{ContestID: &contestID, AuthorID: &userID})
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch questions: %w", err)
	}
	return questions, nil
}

func (s *BaseAPI) AnswerContestQuestion(ctx context.Context, id int, text string) error {
	if err := s.db.AnswerContestQuestion(ctx, id, text); err != nil {
		return fmt.Errorf("couldn't answer question: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateContestAnnouncement(ctx context.Context, announcementID int, text string) error {
	if err := s.db.UpdateContestAnnouncement(ctx, announcementID, text); err != nil {
		return fmt.Errorf("couldn't update announcement: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteContestAnnouncement(ctx context.Context, announcementID int) error {
	if err := s.db.DeleteContestAnnouncement(ctx, announcementID); err != nil {
		return fmt.Errorf("couldn't delete announcement: %w", err)
	}
	return nil
}
