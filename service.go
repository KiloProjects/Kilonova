package kilonova

// TypeServicer is an interface for a provider for UserService, ProblemService, TestService, SubmissionService and SubTestService
type TypeServicer interface {
	UserService() UserService
	ProblemService() ProblemService
	TestService() TestService
	SubmissionService() SubmissionService
	SubTestService() SubTestService
}
