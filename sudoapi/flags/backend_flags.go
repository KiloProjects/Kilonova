package flags

import (
	"github.com/KiloProjects/kilonova/domain/config"
)

var (
	ImportantUpdatesWebhook = config.GenFlag[string]("admin.important_webhook", "", "Webhook URL for audit log-level events")
	VerboseUpdatesWebhook   = config.GenFlag[string]("admin.verbose_webhook", "", "Webhook URL for verbose platform information")

	EmailBranding = config.GenFlag("admin.mailer.branding", "Kilonova", "Branding to use at the end of emails")

	SignupEnabled = config.GenFlag("feature.platform.signup", true, "Manual signup")
)

// DB
var (
	MaxSessionCount = config.GenFlag[int]("behavior.sessions.max_concurrent", 10, "Maximum number of sessions a user can have in total")
)

// captcha
var (
	CaptchaEnabled      = config.GenFlag("feature.captcha.enabled", false, "Enable prompting for CAPTCHAs")
	CaptchaTriggerCount = config.GenFlag("feature.captcha.min_trigger", 10, "Maximum number of sign ups from an ip in 10 minutes before all requests trigger a CAPTCHA")
)

// virtual contests
var (
	NormalUserVirtualContests = config.GenFlag[bool]("behavior.contests.anyone_virtual", false, "Anyone can create virtual contests")
	NormalUserVCLimit         = config.GenFlag[int]("behavior.contests.normal_user_max_day", 10, "Number of maximum contests a non-proposer can create per day")
)

// discord
var (
	// ProposerRoleID = config.GenFlag("integrations.discord.role_ids.proposer", "", "asdf")

	ProblemAnnouncementChannel = config.GenFlag("integrations.discord.publish_announcement_channel", "", "Discord channel to announce new problems")
)

var (
	ExternalResourcesEnabled = config.GenFlag("feature.external_resources.enabled", true, "External resources availability on this instance")
)

var (
	LockdownProblemEditor = config.GenFlag("behavior.problems.lockdown_published", false, "Lockdown published problems such that only admins can edit them")
)

var (
	UserCanChangeNames = config.GenFlag("feature.username_changes.enabled", true, "Anyone can change their usernames")
)

var (
	SubForEveryoneConfig    = config.GenFlag("behavior.everyone_subs", true, "Anyone can view others' source code")
	SubForEveryoneBlacklist = config.GenFlag("behavior.everyone_subs.blacklist", []int{}, "Blacklist of problems where nobody should see eachother's source code")

	PastesEnabled = config.GenFlag("feature.pastes.enabled", true, "Pastes")

	LimitedSubCount = config.GenFlag[int]("behavior.submissions.max_viewing_count", 9999, "Maximum number of submissions to count on subs page. Set to < 0 to disable")
)

var (
	WaitingSubLimit    = config.GenFlag[int]("behavior.submissions.user_max_waiting", 5, "Maximum number of unfinished submissions in the eval queue (for a single user)")
	TotalSubLimit      = config.GenFlag[int]("behavior.submissions.user_max_minute", 20, "Maximum number of submissions uploaded per minute (for a single user with verified email)")
	UnverifiedSubLimit = config.GenFlag[int]("behavior.submissions.user_max_unverified", 5, "Maximum number of submissions uploaded per minute (for a single user with unverified email)")
)

var TestMaxMemKB = config.GenFlag("eval.test_max_mem_kb", 1048576, "Maximum memory limit (KB) a problem may request per test")
