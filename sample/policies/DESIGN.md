- This system will be running in the CICD when the PR is opened, with different levels of enforcement.
- Each level of enforcement will have a different behavior.
  - There are three level enforcement: RECOMMEND, WARNING, BLOCK  
  - RECOMMEND: if services fail the check, a GitHub comment will be added to the Pull Request
  - WARNING: do what RECOMMEND does, and also fail the CI checks
  - BLOCKING: do what WARNING does, but PR cannot be merged if the CI fails


- The policy primarily use Open Policy Agent language (Rego), it can also run additional binary to get the output and then act upon it.
  - Currently support MINT (internal tool to scan k8s manifest for custom organization rules)
  - TODO: coordinate MINT output with the system

- The system consists of 3 steps:
  1. On the Pull Requests, retrieve the services that are being changed. Then, execute `kustomize build` on all overlays `stg`, `prod`, `sandbox`. Each build is running on its own CI via the strategy matrix to parallel it.
  2. Each build will be fed into a CI that loops over our OPA policies (described in the `compliance-config.yaml`), and evaluate the service against those policies. There will be 2 CIs for this, one with Required status and one without it. One with the Required will fail if the policies in BLOCKING level fail. Other level will be ran in the CI without Required status.
  3. Upon finishing, a policy eval report will be generated and attached to the Pull Request. Old report must be removed to avoid spamming. Since the PR can be long, keep the report concise and put more information in the Expandable `<details>` if needed.

