package flags

import "github.com/KiloProjects/kilonova/internal/config"

var (
	CCDisclaimer    = config.GenFlag("frontend.footer.cc_disclaimer", true, "CC disclaimer in footer")
	DiscordInviteID = config.GenFlag("frontend.footer.discord_id", "Qa6Ytgh", "Invite ID for Discord server")

	AllSubsPage       = config.GenFlag("feature.frontend.all_subs_page", true, "Anyone can view all submissions")
	ViewOtherProfiles = config.GenFlag("feature.frontend.view_other_profiles", true, "Allow anyone to view other profiles")

	FrontPageLatestProblems = config.GenFlag("feature.frontend.front_page_latest_pbs", true, "Show list with latest published problems on front page")
	FrontPageProblems       = config.GenFlag("feature.frontend.front_page_pbs", true, "Show problems on front page")
	FrontPagePbDetails      = config.GenFlag("feature.frontend.front_page_pbs_links", true, "On the front page problems, show links to other resources")
	FrontPageRandomProblem  = config.GenFlag("feature.frontend.front_page_random_pb", true, "On the front page problems, show buttons to draw a random problem")

	FrontPageAnnouncement = config.GenFlag("frontend.front_page_announcement", "default", `Custom front page announcement ("default" = default text)`)

	SidebarContests = config.GenFlag("feature.frontend.front_page_csidebar", true, "Show contests in sidebar on the front page")
	ShowTrending    = config.GenFlag("frontend.front_page.show_trending", true, "Show trending problems on the front page sidebar")

	ForceLogin = config.GenFlag("behavior.force_authed", false, "Force authentication when accessing website")

	GoatCounterDomain = config.GenFlag("feature.analytics.goat_prefix", "https://goat.kilonova.ro", "URL prefix for GoatCounter analytics")
	TwiplaID          = config.GenFlag("feature.analytics.twipla_id", "", "ID for TWIPLA Analytics integration")

	NavbarBranding = config.GenFlag("frontend.navbar.branding", "Kilonova", "Branding in navbar")

	FeedbackURL    = config.GenFlag("feature.frontend.feedback_url", "", "Feedback URL for main page")
	QuickSearchBox = config.GenFlag("feature.frontend.quick_search", false, "Quick search box on main page")

	Sentry    = config.GenFlag("feature.frontend.sentry", false, "Enable Sentry error reporting")
	SentryDSN = config.GenFlag("feature.frontend.sentry_dsn", "", "DSN for sentry error reporting")
)

var (
	DonationsEnabled = config.GenFlag("frontend.donations.enabled", true, "Donations page enabled")
	DonationsNag     = config.GenFlag("frontend.donation.frontpage_nag", true, "Donations front page notification")
	PaypalID         = config.GenFlag("frontend.donation.paypal_btn_id", "", "Paypal Donate button ID")
	BuyMeACoffeeName = config.GenFlag("frontend.donation.bmac_name", "", "Name of Buy Me a Coffee page")

	StripeButtonID    = config.GenFlag("frontend.donation.stripe_button_id", "", "Stripe donation button ID")
	StripePK          = config.GenFlag("frontend.donation.stripe_publishable_key", "", "Stripe Publishable Key")
	StripePaymentLink = config.GenFlag("frontend.donation.stripe_payment_link", "", "Stripe donation payment link URL")

	MainPageLogin = config.GenFlag("feature.frontend.main_page_login", false, "Login modal on front page")

	NavbarProblems    = config.GenFlag("feature.frontend.navbar.problems_btn", true, "Navbar button: Problems")
	NavbarContests    = config.GenFlag("feature.frontend.navbar.contests_btn", false, "Navbar button: Contests")
	NavbarSubmissions = config.GenFlag("feature.frontend.navbar.submissions_btn", true, "Navbar button: Submissions")

	FooterTimings = config.GenFlag("feature.frontend.footer.time_statistics", true, "Show measurements about time taken to render pages in footer")

	PinnedProblemList = config.GenFlag("frontend.front_page.pinned_problem_list", 0, "Pinned problem list (front page sidebar)")
	RootProblemList   = config.GenFlag("frontend.front_page.root_problem_list", 0, "Root problem list (front page main content)")
)
