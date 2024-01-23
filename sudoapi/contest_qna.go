package sudoapi

import (
	"context"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
)

func (s *BaseAPI) CreateContestQuestion(ctx context.Context, contest *kilonova.Contest, authorID int, text string) (int, *StatusError) {
	if contest.QuestionCooldown > 0 {
		question, err := s.db.ContestQuestions(ctx, db.QuestionFilter{ContestID: &contest.ID, AuthorID: &authorID})
		if err != nil {
			return -1, WrapError(err, "Could not check for question cooldown")
		}
		if len(question) > 0 {
			if d := contest.QuestionCooldown - time.Since(question[0].AskedAt); d > 0 {
				return -1, Statusf(http.StatusTooManyRequests, "You are going too fast! Please wait %d more second(s) before asking another question.", int(d.Seconds())+1)
			}
		}
	}

	id, err := s.db.CreateContestQuestion(ctx, contest.ID, authorID, text)
	if err != nil {
		return -1, WrapError(err, "Couldn't ask question")
	}
	return id, nil
}

func (s *BaseAPI) CreateContestAnnouncement(ctx context.Context, contestID int, text string) (int, *StatusError) {
	id, err := s.db.CreateContestAnnouncement(ctx, contestID, text)
	if err != nil {
		return -1, WrapError(err, "Couldn't create announcement")
	}
	return id, nil
}

func (s *BaseAPI) ContestAnnouncements(ctx context.Context, contestID int) ([]*kilonova.ContestAnnouncement, *StatusError) {
	announcements, err := s.db.ContestAnnouncements(ctx, contestID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch announcements")
	}
	return announcements, nil
}

func (s *BaseAPI) ContestAnnouncement(ctx context.Context, id int) (*kilonova.ContestAnnouncement, *StatusError) {
	announcement, err := s.db.ContestAnnouncement(ctx, id)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch announcement")
	}
	if announcement == nil {
		return nil, WrapError(ErrNotFound, "Couldn't fetch question")
	}
	return announcement, nil
}

func (s *BaseAPI) ContestQuestions(ctx context.Context, contestID int) ([]*kilonova.ContestQuestion, *StatusError) {
	questions, err := s.db.ContestQuestions(ctx, db.QuestionFilter{ContestID: &contestID})
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch questions")
	}
	return questions, nil
}

func (s *BaseAPI) ContestQuestion(ctx context.Context, id int) (*kilonova.ContestQuestion, *StatusError) {
	question, err := s.db.ContestQuestion(ctx, id)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch question")
	}
	if question == nil {
		return nil, WrapError(ErrNotFound, "Couldn't fetch question")
	}
	return question, nil
}

func (s *BaseAPI) ContestUserQuestions(ctx context.Context, contestID, userID int) ([]*kilonova.ContestQuestion, *StatusError) {
	questions, err := s.db.ContestQuestions(ctx, db.QuestionFilter{ContestID: &contestID, AuthorID: &userID})
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch questions")
	}
	return questions, nil
}

func (s *BaseAPI) AnswerContestQuestion(ctx context.Context, id int, text string) *StatusError {
	if err := s.db.AnswerContestQuestion(ctx, id, text); err != nil {
		return WrapError(err, "Couldn't answer question")
	}
	return nil
}

func (s *BaseAPI) UpdateContestAnnouncement(ctx context.Context, announcementID int, text string) *StatusError {
	if err := s.db.UpdateContestAnnouncement(ctx, announcementID, text); err != nil {
		return WrapError(err, "Couldn't update announcement")
	}
	return nil
}

func (s *BaseAPI) DeleteContestAnnouncement(ctx context.Context, announcementID int) *StatusError {
	if err := s.db.DeleteContestAnnouncement(ctx, announcementID); err != nil {
		return WrapError(err, "Couldn't delete announcement")
	}
	return nil
}
