package sudoapi

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/crypto/bcrypt"

	_ "embed"
)

var (
	CanChangeNames = config.GenFlag("feature.username_changes.enabled", true, "Anyone can change their usernames")
)

func (s *BaseAPI) UserBrief(ctx context.Context, id int) (*kilonova.UserBrief, error) {
	user, err := s.UserFull(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user.UserBrief, nil
}

func (s *BaseAPI) UserFull(ctx context.Context, id int) (*kilonova.UserFull, error) {
	user, err := s.db.User(ctx, kilonova.UserFilter{ID: &id})
	if err != nil || user == nil {
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get user", slog.Any("err", err))
		}
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("user not found: %w", ErrNotFound)
	}
	return user, nil
}

func (s *BaseAPI) UserBriefByName(ctx context.Context, name string) (*kilonova.UserBrief, error) {
	user, err := s.UserFullByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &user.UserBrief, nil
}

func (s *BaseAPI) UserFullByName(ctx context.Context, name string) (*kilonova.UserFull, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, Statusf(400, "Username not specified")
	}
	user, err := s.db.User(ctx, kilonova.UserFilter{Name: &name})
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found: %w", ErrNotFound)
	}
	return user, nil
}

func (s *BaseAPI) UserFullByEmail(ctx context.Context, email string) (*kilonova.UserFull, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, Statusf(400, "Email not specified")
	}
	user, err := s.db.User(ctx, kilonova.UserFilter{Email: &email})
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found: %w", ErrNotFound)
	}
	return user, nil
}

func (s *BaseAPI) UsersBrief(ctx context.Context, filter kilonova.UserFilter) ([]*kilonova.UserBrief, error) {
	users, err := s.db.Users(ctx, filter)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get users", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get users: %w", err)
	}
	return mapUsersBrief(users), nil
}

func mapUsersBrief(users []*kilonova.UserFull) []*kilonova.UserBrief {
	var usersBrief []*kilonova.UserBrief
	for _, user := range users {
		usersBrief = append(usersBrief, user.Brief())
	}
	if len(usersBrief) == 0 {
		return []*kilonova.UserBrief{}
	}
	return usersBrief
}

func (s *BaseAPI) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, error) {
	cnt, err := s.db.CountUsers(ctx, filter)
	if err != nil {
		return -1, fmt.Errorf("couldn't get user count: %w", err)
	}
	return cnt, nil
}

func (s *BaseAPI) UpdateUser(ctx context.Context, userID int, upd kilonova.UserUpdate) error {
	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{UserUpdate: upd})
}

func (s *BaseAPI) updateUser(ctx context.Context, userID int, upd kilonova.UserFullUpdate) error {
	if err := s.db.UpdateUser(ctx, userID, upd); err != nil {
		slog.WarnContext(ctx, "Couldn't update user", slog.Any("err", err))
		return fmt.Errorf("couldn't update user: %w", err)
	}
	sessions, err := s.UserSessions(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get user sessions", slog.Any("err", err))
		return nil
	}
	for _, sess := range sessions {
		s.sessionUserCache.Delete(sess.ID)
	}

	return nil
}

// Since it's a check-update situation, use a mutex to synchronize eventual double updates
var usernameChangeMu sync.Mutex

// fromAdmin also should include the forced username changes
func (s *BaseAPI) UpdateUsername(ctx context.Context, user *kilonova.UserFull, newName string, checkUsed bool, fromAdmin bool) error {
	usernameChangeMu.Lock()
	defer usernameChangeMu.Unlock()

	if !(CanChangeNames.Value() || fromAdmin) {
		return Statusf(401, "Username changes have been disabled by administrator")
	}

	if err := s.CheckValidUsername(newName); err != nil {
		return err
	}

	if !fromAdmin {
		chAt, err := s.db.LastUsernameChange(ctx, user.ID)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get last username change date", slog.Any("err", err))
			return fmt.Errorf("couldn't get last change date: %w", err)
		}
		var nextEligibleDate time.Time
		if !chAt.Equal(user.CreatedAt) {
			nextEligibleDate = chAt.Add(14 * 24 * time.Hour)
		}
		if nextEligibleDate.After(time.Now()) {
			return Statusf(400, "You can only change your username at most once every 14 days. You may change it again on %s", nextEligibleDate.Format(time.DateTime))
		}
	}

	if _, err := s.UserBriefByName(ctx, newName); err == nil {
		return Statusf(400, "Name must not be used by anyone currently")
	}

	if checkUsed {
		used, err := s.db.NameUsedBefore(ctx, newName)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't check if name was used before", slog.Any("err", err))
		}
		if used {
			userIDs, err := s.db.HistoricalUsernameHolders(ctx, newName)
			if err != nil && !errors.Is(err, context.Canceled) {
				userIDs = []int{-1, -2}
			}
			if !(len(userIDs) == 0 || (len(userIDs) == 1 && userIDs[0] == user.ID)) {
				return Statusf(400, "New name must have never been used by anyone else. Contact us on Discord if you want to take the name for yourself.")
			}
		}
	}

	f := false
	if err := s.updateUser(ctx, user.ID, kilonova.UserFullUpdate{Name: &newName, NameChangeRequired: &f}); err != nil {
		return err
	}

	s.LogInfo(ctx, "Username was changed", slog.Any("user", user), slog.String("new_name", newName))
	return nil
}

func (s *BaseAPI) SetForceUsernameChange(ctx context.Context, userID int, force bool) error {
	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{NameChangeRequired: &force})
}
func (s *BaseAPI) SetUserLockout(ctx context.Context, userID int, lockout bool) error {
	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{LockedLogin: &lockout})
}

func (s *BaseAPI) UsernameChangeHistory(ctx context.Context, userID int) ([]*kilonova.UsernameChange, error) {
	changes, err := s.db.UsernameChangeHistory(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get username change history", slog.Any("err", err))
		return []*kilonova.UsernameChange{}, fmt.Errorf("couldn't get change history: %w", err)
	}
	return changes, nil
}

// Returns the people that last held the given username
// Used for redirecting on the frontend the profile page
func (s *BaseAPI) HistoricalUsernameHolder(ctx context.Context, name string) (*kilonova.UserBrief, error) {
	userIDs, err := s.db.HistoricalUsernameHolders(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get username holders: %w", err)
	}
	if len(userIDs) == 0 {
		return nil, nil
	}
	return s.UserBrief(ctx, userIDs[0])
}

func (s *BaseAPI) VerifyUserPassword(ctx context.Context, uid int, password string) error {
	hashedPassword, err := s.db.HashedPassword(ctx, uid)
	if err != nil || hashedPassword == "" {
		return fmt.Errorf("user not found: %w", ErrNotFound)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return Statusf(400, "Invalid password")
	} else if err != nil {
		return ErrUnknownError
	}

	return nil
}

func (s *BaseAPI) DeleteUser(ctx context.Context, user *kilonova.UserBrief) error {
	if err := s.db.DeleteUser(ctx, user.ID); err != nil {
		return fmt.Errorf("couldn't delete user: %w", err)
	}
	s.LogUserAction(ctx, "Deleted user", slog.Any("user", user))
	return nil
}

func (s *BaseAPI) UpdateUserPassword(ctx context.Context, uid int, password string) error {
	if err := s.CheckValidPassword(password); err != nil {
		return err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("couldn't generate hash: %w", err)
	}

	// TODO: Replace UpdateUserPasswordHash with a key in UserFullUpdate
	if err := s.db.UpdateUserPasswordHash(ctx, uid, hash); err != nil {
		return fmt.Errorf("couldn't update password: %w", err)
	}
	return nil
}

// TODO: displayName probably doesn't have to be *string, can be just string, but this was implemented quickly
func (s *BaseAPI) GenerateUser(ctx context.Context, uname, pwd, lang string, theme kilonova.PreferredTheme, displayName *string, email *string, bio string) (*kilonova.UserFull, error) {
	uname = strings.TrimSpace(uname)
	if err := s.CheckValidUsername(uname); err != nil {
		return nil, err
	}
	if err := s.CheckValidPassword(pwd); err != nil {
		return nil, err
	}
	if !(lang == "" || lang == "en" || lang == "ro") {
		return nil, Statusf(400, "Invalid language.")
	}
	if !(theme == kilonova.PreferredThemeNone || theme == kilonova.PreferredThemeLight || theme == kilonova.PreferredThemeDark) {
		return nil, Statusf(400, "Invalid theme.")
	}

	if exists, err := s.db.UserExists(ctx, uname, "INVALID_EMAIL"); err != nil || exists {
		return nil, Statusf(400, "User matching username already exists!")
	}

	if lang == "" {
		lang = config.Common.DefaultLang
	}
	if theme == kilonova.PreferredThemeNone {
		theme = kilonova.PreferredThemeDark
	}

	if email == nil {
		// Dummy email
		genEmail := fmt.Sprintf("email_%s@kilonova.ro", uname)
		email = &genEmail
	}

	dName := ""
	if displayName != nil && len(*displayName) > 0 {
		dName = *displayName
	}

	id, err := s.createUser(ctx, uname, *email, pwd, lang, theme, dName, bio, true)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't create user", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't create user: %w", err)
	}

	user, err := s.UserFull(ctx, id)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get full user", slog.Any("err", err))
	}

	return user, err
}

//go:embed emails/generated.html
var generatedUserEmail string
var generatedUserTempl = template.Must(template.New("emailTempl").Parse(generatedUserEmail))

// Basically [a-zA-Z0-9] but exclude i/I/l/L and 0/o/O since they may be easily mistaken
const userPasswordAlphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ123456789"

type UserGenerationRequest struct {
	Name     string `json:"username"`
	Password string `json:"password"`
	Lang     string `json:"language"`

	Bio string `json:"bio"`

	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`

	ContestID *int `json:"contest_id"`

	PasswordByMail bool `json:"password_by_mail"`
	// PasswordByMailTo overrides whom to send the email to
	PasswordByMailTo *string `json:"password_by_mail_to"`

	MailSubject *string `json:"mail_subject"`
}

func (s *BaseAPI) RandomPassword() string {
	return kilonova.RandomStringChars(7, userPasswordAlphabet)
}

// returns password, UserFull and eventual error
func (s *BaseAPI) GenerateUserFlow(ctx context.Context, args UserGenerationRequest) (string, *kilonova.UserFull, error) {

	if args.PasswordByMail {
		if !s.MailerEnabled() {
			return "", nil, kilonova.Statusf(400, "Mailer has been disabled, but sending password by email was enabled.")
		}
		if args.Email == nil && args.PasswordByMailTo == nil {
			return "", nil, kilonova.Statusf(400, "Cannot send password by email if no address was given")
		}
	}

	if args.Password == "" {
		args.Password = s.RandomPassword()
	}

	var contest *kilonova.Contest
	if args.ContestID != nil && *args.ContestID > 0 {
		contest2, err := s.Contest(ctx, *args.ContestID)
		if err != nil {
			return "", nil, err
		}
		contest = contest2
	}

	user, err := s.GenerateUser(ctx, args.Name, args.Password, args.Lang, kilonova.PreferredThemeDark, args.DisplayName, args.Email, args.Bio)
	if err != nil {
		return "", nil, err
	}

	if contest != nil {
		if err := s.RegisterContestUser(ctx, contest, user.ID, nil, true); err != nil {
			return args.Password, user, err
		}
	}

	if args.PasswordByMail {
		emailArgs := struct {
			Name       string
			Username   string
			Password   string
			Contest    *kilonova.Contest
			HostPrefix string
			Branding   string
		}{
			Name:       user.Name,
			Username:   user.Name,
			Password:   args.Password,
			Contest:    contest,
			HostPrefix: config.Common.HostPrefix,
			Branding:   EmailBranding.Value(),
		}
		if user.DisplayName != "" {
			emailArgs.Name = user.DisplayName
		}
		if val, ok := config.GetFlagVal[string]("frontend.navbar.branding"); ok && len(val) > 0 {
			emailArgs.Branding = val
		}
		var b bytes.Buffer
		if err := generatedUserTempl.ExecuteTemplate(&b, user.PreferredLanguage, emailArgs); err != nil {
			slog.ErrorContext(ctx, "Error rendering password send email", slog.Any("err", err))
			return args.Password, user, fmt.Errorf("could not render email: %w", err)
		}
		var sendTo string
		if args.Email != nil {
			sendTo = *args.Email
		}
		if args.PasswordByMailTo != nil {
			sendTo = *args.PasswordByMailTo
		}

		var subject = kilonova.GetText(user.PreferredLanguage, "mail.subject.generated")
		if args.MailSubject != nil {
			subject = *args.MailSubject
		}

		if err := s.SendMail(ctx, &kilonova.MailerMessage{
			To:          sendTo,
			Subject:     subject,
			HTMLContent: b.String(),
		}); err != nil {
			slog.WarnContext(ctx, "Could not send email", slog.Any("err", err))
			return args.Password, user, err
		}
	}

	return args.Password, user, nil
}

func (s *BaseAPI) createUser(ctx context.Context, username, email, password, lang string, theme kilonova.PreferredTheme, displayName string, bio string, generated bool) (int, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return -1, err
	}

	id, err := s.db.CreateUser(ctx, username, hash, email, lang, theme, displayName, bio, generated)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't create user", slog.Any("err", err))
		return -1, err
	}

	if id == 1 {
		var True = true
		if err := s.updateUser(ctx, id, kilonova.UserFullUpdate{Admin: &True, Proposer: &True}); err != nil {
			slog.WarnContext(ctx, "Couldn't set first user as admin", slog.Any("err", err))
			return id, err
		}
	}

	return id, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), err
}

func getGravatar(ctx context.Context, email string, size int) (io.ReadSeekCloser, time.Time, error) {
	v := url.Values{}
	v.Add("s", strconv.Itoa(size))
	v.Add("d", "identicon")
	bSum := md5.Sum([]byte(email))

	req, _ := http.NewRequestWithContext(context.WithoutCancel(ctx), "GET", fmt.Sprintf("https://gravatar.com/avatar/%s.png?%s", hex.EncodeToString(bSum[:]), v.Encode()), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return nil, time.Unix(0, 0), err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") == "text/html" {
		return nil, time.Unix(0, 0), Statusf(resp.StatusCode, "Invalid gravatar response")
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, time.Unix(0, 0), err
	}
	time, _ := http.ParseTime(resp.Header.Get("last-modified"))
	return &bytesReaderCloser{bytes.NewReader(buf)}, time, nil
}

func gravatarBucketName(email string, size int) string {
	bSum := sha256.Sum256([]byte(email))
	return fmt.Sprintf("%s-%d.png", hex.EncodeToString(bSum[:]), size)
}

// if r is nil, it fetches the gravatar from the web
func (s *BaseAPI) saveGravatar(ctx context.Context, email string, size int, r io.Reader) error {
	if r != nil {
		return s.avatarBucket.WriteFile(gravatarBucketName(email, size), r, 0644)
	}

	r, _, err := getGravatar(ctx, email, size)
	if err != nil {
		return err
	}
	return s.avatarBucket.WriteFile(gravatarBucketName(email, size), r, 0644)
}

// valid is true only if maxLastMod is greater than the saved value and if the avatar is saved
func (s *BaseAPI) avatarFromBucket(filename string, maxLastMod time.Time) (io.ReadSeekCloser, time.Time, bool, error) {
	f, err := s.avatarBucket.ReadSeeker(filename)
	if err != nil {
		return nil, time.Unix(0, 0), false, err
	}
	stat, err := s.avatarBucket.Stat(filename)
	if err != nil {
		f.Close()
		return nil, time.Unix(0, 0), false, err
	}
	if stat.ModTime().Before(maxLastMod) {
		return f, stat.ModTime(), false, nil
	}
	return f, stat.ModTime(), true, nil
}

// if manager.GetGravatar errors out or is not valid, it fetches the gravatar from the web
func (s *BaseAPI) GetGravatar(ctx context.Context, email string, size int, maxLastMod time.Time) (io.ReadSeekCloser, time.Time, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if r, t, valid, err := s.avatarFromBucket(gravatarBucketName(email, size), maxLastMod); valid && err == nil {
		return r, t, valid, err
	}
	r, t, err := getGravatar(ctx, email, size)
	if err != nil {
		return r, t, false, err
	}
	if err := s.saveGravatar(ctx, email, size, r); err != nil {
		slog.WarnContext(ctx, "Could not save avatar", slog.Any("err", err))
	}
	r.Seek(0, io.SeekStart)
	return r, t, true, nil
}

func discordAvatarBucketName(user *kilonova.UserFull, size int) string {
	bSum := sha256.Sum256([]byte(*user.DiscordID))
	return fmt.Sprintf("%s-%d.png", hex.EncodeToString(bSum[:]), size)
}

func getDiscordAvatar(ctx context.Context, dUser *discordgo.User, size int) (io.ReadSeekCloser, time.Time, error) {
	req, _ := http.NewRequestWithContext(context.WithoutCancel(ctx), "GET", dUser.AvatarURL(strconv.Itoa(size)), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return nil, time.Unix(0, 0), err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") == "text/html" {
		return nil, time.Unix(0, 0), Statusf(resp.StatusCode, "Invalid discord avatar response")
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, time.Unix(0, 0), err
	}
	time, _ := http.ParseTime(resp.Header.Get("last-modified"))
	return &bytesReaderCloser{bytes.NewReader(buf)}, time, nil
}

// if r is nil, it fetches the gravatar from the web
func (s *BaseAPI) saveDiscordAvatar(ctx context.Context, user *kilonova.UserFull, dUser *discordgo.User, size int, r io.Reader) error {
	if r != nil {
		return s.avatarBucket.WriteFile(discordAvatarBucketName(user, size), r, 0644)
	}

	r, _, err := getDiscordAvatar(ctx, dUser, size)
	if err != nil {
		return err
	}
	return s.avatarBucket.WriteFile(discordAvatarBucketName(user, size), r, 0644)
}

// if manager.GetDiscordAvatar errors out or is not valid, it fetches the gravatar from the web
func (s *BaseAPI) GetDiscordAvatar(ctx context.Context, user *kilonova.UserFull, size int, maxLastMod time.Time) (io.ReadSeekCloser, time.Time, bool, error) {
	if user.DiscordID == nil {
		return nil, time.Time{}, false, nil
	}
	if r, t, valid, err := s.avatarFromBucket(discordAvatarBucketName(user, size), maxLastMod); valid && err == nil {
		return r, t, valid, err
	}

	dUser, err := s.GetDiscordIdentity(ctx, user.ID)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	if dUser == nil {
		return nil, time.Time{}, false, nil
	}

	r, t, err := getDiscordAvatar(ctx, dUser, size)
	if err != nil {
		return r, t, false, err
	}
	if err := s.saveDiscordAvatar(ctx, user, dUser, size, r); err != nil {
		slog.WarnContext(ctx, "Could not save avatar", slog.Any("err", err))
	}
	r.Seek(0, io.SeekStart)
	return r, t, true, nil
}

type bytesReaderCloser struct{ *bytes.Reader }

func (r *bytesReaderCloser) Close() error { return nil }
