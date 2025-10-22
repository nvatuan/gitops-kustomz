package runner

import "github.com/gh-nvat/gitops-kustomz/src/pkg/models"

type RunnerInterface interface {
	// Initialize the runner with necessary context and data
	Initialize() error

	// Build manifests for a specific environment
	BuildManifests() (*models.BuildManifestResult, error)

	// Build manifests for a specific environment
	DiffManifests(*models.BuildManifestResult) (map[string]models.EnvironmentDiff, error)

	// Main routine to process the runner
	Process() error

	// Handling the export
	Output(data *models.ReportData) error
}
