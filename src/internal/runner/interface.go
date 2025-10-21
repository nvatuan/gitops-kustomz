package runner

type Runner struct {
	RunMode  string
	Instance RunnerInterface
}

type RunnerInterface interface {
	// Run some initialization steps to populate the data needed for the run mode
	// Eg. in GitHub mode, this step will attempt fetching the PR infos
	// This step should be run in go-routine to avoid blocking the main thread
	Initialize() error
}
