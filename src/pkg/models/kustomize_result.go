package models

type BuildManifestResult struct {
	EnvManifestBuild map[string]BuildEnvManifestResult
}

type BuildEnvManifestResult struct {
	Environment    string
	BeforeManifest []byte
	AfterManifest  []byte
}

type PolicyEvaluateResult struct {
	EnvPolicyEvaluate map[string]PolicyEnvEvaluateResult
}

type PolicyEnvEvaluateResult struct {
	Environment string

	// if key exists, value is not empty => failed
	// if key exists, value empty => passed
	PolicyIdToEvalFailMsgs map[string][]string
}
