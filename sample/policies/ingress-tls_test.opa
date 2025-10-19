package kustomization.tls_test

import rego.v1

# Test for Ingress without TLS
test_ingress_no_tls if {
	count(kustomization.tls.deny) == 1 with input as {
		"request": {
			"kind": {"kind": "Ingress"},
			"object": {
				"apiVersion": "networking.k8s.io/v1",
				"kind": "Ingress",
				"metadata": {"name": "test-ingress"},
				"spec": {
					"rules": [{
						"host": "example.com",
						"http": {"paths": [{"path": "/", "backend": {"service": {"name": "test"}}}]},
					}],
				},
			},
		},
	}
}

# Test for Ingress with TLS
test_ingress_with_tls if {
	count(kustomization.tls.deny) == 0 with input as {
		"request": {
			"kind": {"kind": "Ingress"},
			"object": {
				"apiVersion": "networking.k8s.io/v1",
				"kind": "Ingress",
				"metadata": {"name": "test-ingress"},
				"spec": {
					"tls": [{
						"hosts": ["example.com"],
						"secretName": "test-tls",
					}],
					"rules": [{
						"host": "example.com",
						"http": {"paths": [{"path": "/", "backend": {"service": {"name": "test"}}}]},
					}],
				},
			},
		},
	}
}

