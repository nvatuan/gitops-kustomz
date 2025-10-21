```bash
❯ conftest test --all-namespaces --combine --policy policies/ha.rego prod.yaml
FAIL - Combined - main - Deployment 'prod-my-app' must have PodAntiAffinity with topologyKey 'kubernetes.io/hostname' for high availability
FAIL - Combined - main - Deployment 'prod-my-app' must have at least 2 replicas for high availability, found: 1

❯ conftest verify --policy policies/ha.rego --policy policies/ha_test.rego

8 tests, 8 passed, 0 warnings, 0 failures, 0 exceptions, 0 skipped
```

 * use `--combine`, inputs shape also change so be careful