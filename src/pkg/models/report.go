package models

import "time"

// ReportData represents the complete report data structure
type ReportData struct {
	Service      string
	Timestamp    time.Time
	BaseCommit   string
	HeadCommit   string
	Environments []string

	// Manifest changes per environment
	ManifestChanges map[string]EnvironmentDiff

	// Policy evaluation results
	PolicyEvaluation PolicyEvaluation
}

// EnvironmentDiff represents diff data for a single environment
type EnvironmentDiff struct {
	LineCount        int
	AddedLineCount   int
	DeletedLineCount int
	Content          string
}

// PolicyEvaluationSummary represents the overall policy evaluation results
type PolicyEvaluation struct {
	// Summary table: Environment -> Success/Failed/Errored counts
	EnvironmentSummary map[string]PolicyCounts

	// Detailed policy matrix
	PolicyMatrix map[string]PolicyMatrix
}

// PolicyCounts represents the count of policies by status for an environment
type PolicyCounts struct {
	Success int
	Failed  int
	Errored int
}

// PolicyMatrix represents the detailed policy evaluation matrix
type PolicyMatrix struct {
	// Policies grouped by enforcement level
	BlockingPolicies    []PolicyResult
	WarningPolicies     []PolicyResult
	RecommendPolicies   []PolicyResult
	OverriddenPolicies  []PolicyResult
	NotInEffectPolicies []PolicyResult
}

// PolicyResult represents the result of a single policy evaluation
type PolicyResult struct {
	PolicyName   string
	Enforcement  string // "BLOCKING", "WARNING", "RECOMMEND"
	Status       string // "Overridden", "Not In Effect", etc.
	FailMessages []string
}

// ReportTemplateData represents the data structure for template rendering
type ReportTemplateData struct {
	ReportData
	RenderedMarkdown string
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
