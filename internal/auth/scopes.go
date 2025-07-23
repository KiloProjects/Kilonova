package auth

var Scopes = []string{
	// api enables access to the API
	"api",

	// admin allows performing admin actions
	// TODO: strip scope from non-admins
	"admin",
	// proposer allows performing proposer actions
	// TODO: strip scope from non-proposers
	"proposer",
}
