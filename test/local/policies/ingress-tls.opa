package kustomization.tls

import rego.v1

# Ingress TLS Policy
# Ensures Ingress resources have TLS configuration

# Check if Ingress has TLS configured
deny contains msg if {
	input.request.kind.kind == "Ingress"
	not has_tls(input.request.object)
	msg := sprintf("Ingress '%s' must have TLS configuration", [input.request.object.metadata.name])
}

# Helper function to check if Ingress has TLS
has_tls(ingress) if {
	ingress.spec.tls
	count(ingress.spec.tls) > 0
}
