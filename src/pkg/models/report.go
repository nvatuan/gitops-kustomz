package models

import "time"

// ReportData represents the complete report data structure
type ReportData struct {
	Service      string    `json:"service"`
	Timestamp    time.Time `json:"timestamp"`
	BaseCommit   string    `json:"baseCommit"`
	HeadCommit   string    `json:"headCommit"`
	Environments []string  `json:"environments"`

	// Manifest changes per environment
	ManifestChanges map[string]EnvironmentDiff `json:"manifestChanges"`

	// Policy evaluation results
	PolicyEvaluation PolicyEvaluation `json:"policyEvaluation"`
}

// EnvironmentDiff represents diff data for a single environment
type EnvironmentDiff struct {
	LineCount        int    `json:"lineCount"`
	AddedLineCount   int    `json:"addedLineCount"`
	DeletedLineCount int    `json:"deletedLineCount"`
	Content          string `json:"content"`
}

// PolicyEvaluationSummary represents the overall policy evaluation results
type PolicyEvaluation struct {
	// Summary table: Environment -> Success/Failed/Errored counts
	EnvironmentSummary map[string]EnvironmentSummaryEnv `json:"environmentSummary"`

	// Detailed policy matrix
	PolicyMatrix map[string]PolicyMatrix `json:"policyMatrix"`
}

type EnvironmentSummaryEnv struct {
	PassingStatus EnforcementPassingStatus `json:"passingStatus"`
	PolicyCounts  PolicyCounts             `json:"policyCounts"`
}

type EnforcementPassingStatus struct {
	PassBlockingCheck  bool `json:"passBlockingCheck"`
	PassWarningCheck   bool `json:"passWarningCheck"`
	PassRecommendCheck bool `json:"passRecommendCheck"`
}

type PolicyCounts struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Omitted int `json:"omitted"`
}

// PolicyMatrix represents the detailed policy evaluation matrix
type PolicyMatrix struct {
	// Policies grouped by enforcement level
	BlockingPolicies    []PolicyResult `json:"blockingPolicies"`
	WarningPolicies     []PolicyResult `json:"warningPolicies"`
	RecommendPolicies   []PolicyResult `json:"recommendPolicies"`
	OverriddenPolicies  []PolicyResult `json:"overriddenPolicies"`
	NotInEffectPolicies []PolicyResult `json:"notInEffectPolicies"`
}

// PolicyResult represents the result of a single policy evaluation
type PolicyResult struct {
	PolicyId     string   `json:"policyId"`
	PolicyName   string   `json:"policyName"`
	IsPassing    bool     `json:"isPassing"` // true or false, if false, FailMessages is not empty
	FailMessages []string `json:"failMessages"`
}

// ReportTemplateData represents the data structure for template rendering
type ReportTemplateData struct {
	ReportData
	RenderedMarkdown string `json:"renderedMarkdown"`
}

/* sample of desired report

<!-- gitops-kustomz: auto-generated comment, please do not remove -->

# 🔍 GitOps Policy Check: my-app

| Timestamp | Base | Head | Environments |
|-|-|-|-
2025-10-22 00:31:12 UTC | local | local | `stg`, `prod`

## 📊 Manifest Changes

### [`stg`]: `8` lines (4➕/4➖)

<details open>
<summary>Click to expand stg diff</summary>

```diff
--- base	2025-10-22 00:31:12
+++ head	2025-10-22 00:31:12
@@ -50,7 +50,7 @@
         - name: ENVIRONMENT
           value: staging
         - name: LOG_LEVEL
-          value: debug
+          value: warn
         image: nginx:1.21
         livenessProbe:
         httpGet:
@@ -69,8 +69,8 @@
           periodSeconds: 5
         resources:
           limits:
-            cpu: 500m
-            memory: 256Mi
+            cpu: 800m
+            memory: 512Mi
           requests:
             cpu: 250m
             memory: 128Mi
@@ -138,7 +138,7 @@
     replicas: 1
   idleReplicaCount: 0
   maxReplicaCount: 8
-  minReplicaCount: 1
+  minReplicaCount: 4
   pollingInterval: 15
   scaleTargetRef:
     name: my-app

```

</details>




### [`prod`]: `6` lines (3➕/3➖)


<details open>
<summary>Click to expand prod diff</summary>

```diff
--- base	2025-10-22 00:31:12
+++ head	2025-10-22 00:31:12
@@ -51,7 +51,7 @@
           value: production
         - name: LOG_LEVEL
           value: info
-        image: nginx:1.21
+        image: nginx:latest
         livenessProbe:
         failureThreshold: 3
         httpGet:
@@ -197,7 +197,7 @@
   namespace: my-app-prod
 spec:
   rules:
-  - host: my-app-prod.example.com
+  - host: my-app.example.com
     http:
       paths:
       - backend:
@@ -209,5 +209,5 @@
         pathType: Prefix
   tls:
   - hosts:
-    - my-app-prod.example.com
+    - my-app.example.com
     secretName: my-app-prod-tls

```

</details>

## 🛡️ Policy Evaluation

| Environments | Success | Failed | Not In Effect or Overridden |
|--------------|---------|--------|------------------------------|
| `prod` | 3 | 1 | 2 |
| `stg` | 3 | 1 | 2 |


<details open> <summary> Policy Evaluation Matrix </summary>

<details> <summary> 1 fail policies that are blocking merge</summary>

| Policy | Env | Fail Message |
|--------|-------------|--------|
| Service High Availability | stg: ❌ | Deployment 'my-app' must have at least 2 replicas |

</details>

<details> <summary> 1 fail policy that is warn level </summary>

| Policy | Env | Fail Message |
|--------|-------------|--------|
| Service Taggings | prod: ❌ | Deployment 'my-app' must have standard taggings |

</details>

<!-- since recommend is empty, we don't render it -->

<details> <summary> 3 failing policies that were overridden or not in effect</summary>

| Policy | Status | Env | Fail Message |
|--------|--------|-----|--------|
| Service Check 1 | Overriden 		| stg: ❌ | Deployment 'my-app' must not have cpu limit |
| Service Check 1 | Overriden 		| prod: ❌ | Deployment 'my-app' must not have cpu limit |
| Service Check 2 | Not In Effect | stg: ❌ | Deployment 'my-app' must have memory requests equals to limits |
| Service Check 2 | Not In Effect | prod: ❌ | Deployment 'my-app' must have memory requests equals to limits |

</details>

Other than that, `6` policies were successful.

*/
