package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) filterUserAgent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if flags.FilterUserAgent.Value() && (util.UserBrief(r) == nil || !util.UserBrief(r).Admin) {
			// If filtering is enabled and user is not admin, disallow common software for bots
			if strings.Contains(r.Header.Get("User-Agent"), "python") {
				errorData(w, "Request blocked", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeVisitor is middleware to make sure the user creating the request is not authenticated
func (s *API) MustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.UserBrief(r).IsAuthed() {
			errorData(w, "You must not be logged in to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAdmin is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdmin(next http.Handler) http.Handler {
	return s.MustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.UserBrief(r).IsAdmin() {
			errorData(w, "You must be an admin to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// MustBeAuthed is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.UserBrief(r).IsAuthed() {
			errorData(w, "You must be authenticated to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeProposer is middleware to make sure the user creating the request is a proposer
func (s *API) MustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.UserBrief(r).IsProposer() {
			errorData(w, "You must be a proposer to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetupSession adds the user with the specified user ID to context
func (s *API) SetupSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.base.SessionUser(r.Context(), getAuthHeader(r), r)
		if err != nil || user == nil {
			if err != nil {
				slog.WarnContext(r.Context(), "Error getting session user", slog.Any("err", err))
			}
			next.ServeHTTP(w, r)
			return
		}
		trace.SpanFromContext(r.Context()).SetAttributes(attribute.Int("user.id", user.ID), attribute.String("user.name", user.Name))
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AuthedUserKey, user)))
	})
}

func (s *API) validateProblemEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
			errorData(w, "You must be authorized to access internal problem data", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *API) validateContestParticipant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.CanSubmitInContest(util.UserBrief(r), util.Contest(r)) {
			errorData(w, "You must be registered and during a contest to do this", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (s *API) validateContestEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.Contest(r).IsEditor(util.UserBrief(r)) {
			errorData(w, "You must be authorized to access this contest data", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (s *API) validateContestVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.IsContestVisible(util.UserBrief(r), util.Contest(r)) {
			errorData(w, "You are not allowed to access this contest", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (s *API) validateProblemVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.IsProblemVisible(util.UserBrief(r), util.Problem(r)) {
			errorData(w, "You are not allowed to access this problem", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (s *API) validateVisibleTests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.CanViewTests(util.UserBrief(r), util.Problem(r)) {
			errorData(w, "You are not allowed to access this problem's tests", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (s *API) validateProblemFullyVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.IsProblemFullyVisible(util.UserBrief(r), util.Problem(r)) {
			errorData(w, "You are not allowed to access this problem data", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (s *API) validateBlogPostVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.IsBlogPostVisible(util.UserBrief(r), util.BlogPost(r)) {
			errorData(w, "You are not allowed to access this post", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateTestID pre-emptively returns if there isnt a valid test ID in the URL params
// Also, it fetches the test from the DB and makes sure it exists
// NOTE: This does not fetch the test data from disk
func (s *API) validateTestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testID, err := strconv.Atoi(r.PathValue("tID"))
		if err != nil {
			errorData(w, "invalid test ID", http.StatusBadRequest)
			return
		}
		test, err := s.base.Test(r.Context(), util.Problem(r).ID, testID)
		if err != nil {
			errorData(w, "test does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TestKey, test)))
	})
}

// TODO: restrucutre validateAttachmentID and validateAttachmentName to use *AttachmentFilter (reduce code repetition)
func (s *API) validateAttachmentID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attID, err := strconv.Atoi(r.PathValue("aID"))
		if err != nil {
			errorData(w, "invalid attachment ID", http.StatusBadRequest)
			return
		}
		if util.Problem(r) == nil && util.BlogPost(r) == nil {
			slog.ErrorContext(r.Context(), "Attachment context is not available")
			return
		}

		var rezAtt *kilonova.Attachment
		if util.Problem(r) != nil {
			att, err := s.base.ProblemAttachment(r.Context(), util.Problem(r).ID, attID)
			if err != nil {
				errorData(w, "attachment does not exist", http.StatusBadRequest)
				return
			}
			if att.Private && !s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
				errorData(w, "you cannot access attachment data!", http.StatusBadRequest)
				return
			}
			rezAtt = att
		} else if util.BlogPost(r) != nil {
			att, err := s.base.BlogPostAttachment(r.Context(), util.BlogPost(r).ID, attID)
			if err != nil {
				errorData(w, "attachment does not exist", http.StatusBadRequest)
				return
			}
			if att.Private && !s.base.IsBlogPostEditor(util.UserBrief(r), util.BlogPost(r)) {
				errorData(w, "you cannot access attachment data!", http.StatusBadRequest)
				return
			}
			rezAtt = att
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AttachmentKey, rezAtt)))
	})
}

// TODO: restrucutre validateAttachmentID and validateAttachmentName to use *AttachmentFilter (reduce code repetition)
func (s *API) validateAttachmentName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attName := r.PathValue("aName")
		if util.Problem(r) == nil && util.BlogPost(r) == nil {
			slog.ErrorContext(r.Context(), "Attachment context is not available")
			return
		}

		var rezAtt *kilonova.Attachment
		if util.Problem(r) != nil {
			att, err := s.base.ProblemAttByName(r.Context(), util.Problem(r).ID, attName)
			if err != nil {
				errorData(w, "attachment does not exist", http.StatusBadRequest)
				return
			}
			if att.Private && !s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
				errorData(w, "you cannot access attachment data!", http.StatusBadRequest)
				return
			}
			rezAtt = att
		} else if util.BlogPost(r) != nil {
			att, err := s.base.BlogPostAttByName(r.Context(), util.BlogPost(r).ID, attName)
			if err != nil {
				errorData(w, "attachment does not exist", http.StatusBadRequest)
				return
			}
			if att.Private && !s.base.IsBlogPostEditor(util.UserBrief(r), util.BlogPost(r)) {
				errorData(w, "you cannot access attachment data!", http.StatusBadRequest)
				return
			}
			rezAtt = att
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AttachmentKey, rezAtt)))
	})
}

func (s *API) validateUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.PathValue("cUID"))
		if err != nil {
			errorData(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		user, err := s.base.UserFull(r.Context(), userID)
		if err != nil {
			errorData(w, "User was not found", http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ContentUserKey, user)))
	})
}

func (s *API) selfOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ContentUser must not be nil, requesting user must be authenticated
		// and the requesting user must EITHER be an admin or the user that is being operated on
		if util.ContentUserBrief(r) == nil || !util.UserBrief(r).IsAuthed() || !(util.UserBrief(r).IsAdmin() || util.ContentUserBrief(r).ID == util.UserBrief(r).ID) {
			errorData(w, "You aren't allowed to do this!", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *API) validateUsername(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.base.UserFullByName(r.Context(), strings.TrimSpace(r.PathValue("cUName")))
		if err != nil {
			errorData(w, "User was not found", http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ContentUserKey, user)))
	})
}

func (s *API) validateBucket(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := datastore.BucketType(r.PathValue("bname"))
		bucket, err := s.base.DataStore().Get(name)
		if err != nil {
			errorData(w, "Invalid bucket", 400)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.BucketKey, bucket)))
	})
}

func (s *API) authedContentUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.UserFull(r) == nil {
			slog.WarnContext(r.Context(), "authedContentUser got nil UserFull in context")
			errorData(w, "User was not found", http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ContentUserKey, util.UserFull(r))))
	})
}

func (s *API) validateBlogPostID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bpID, err := strconv.Atoi(r.PathValue("bpID"))
		if err != nil {
			errorData(w, "invalid blog post ID", http.StatusBadRequest)
			return
		}
		post, err := s.base.BlogPost(r.Context(), bpID)
		if err != nil {
			errorData(w, "blog post does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.BlogPostKey, post)))
	})
}

func (s *API) validateBlogPostName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bpName := strings.TrimSpace(r.PathValue("bpName"))
		post, err := s.base.BlogPostBySlug(r.Context(), bpName)
		if err != nil {
			errorData(w, "blog post does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.BlogPostKey, post)))
	})
}

// validateProblemID pre-emptively returns if there isn't a valid problem ID in the URL params
// Also, it fetches the problem from the DB and makes sure it exists
func (s *API) validateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.Atoi(r.PathValue("problemID"))
		if err != nil {
			errorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		problem, err := s.base.Problem(r.Context(), problemID)
		if err != nil {
			errorData(w, "problem does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemKey, problem)))
	})
}

// validateProblemListID pre-emptively returns if there isn't a valid problem list ID in the URL params
// Also, it fetches the problem from the DB and makes sure it exists
func (s *API) validateProblemListID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pblistID, err := strconv.Atoi(r.PathValue("pblistID"))
		if err != nil {
			errorData(w, "invalid problem list ID", http.StatusBadRequest)
			return
		}
		pblist, err := s.base.ProblemList(r.Context(), pblistID)
		if err != nil {
			errorData(w, "problem list does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemListKey, pblist)))
	})
}

func (s *API) validateContestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contestID, err := strconv.Atoi(r.PathValue("contestID"))
		if err != nil {
			errorData(w, "invalid contest ID", http.StatusBadRequest)
			return
		}
		contest, err := s.base.Contest(r.Context(), contestID)
		if err != nil {
			errorData(w, "contest does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ContestKey, contest)))
	})
}

func getAuthHeader(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "guest" {
		header = ""
	}
	return header
}
