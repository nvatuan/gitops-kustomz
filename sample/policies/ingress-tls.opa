package kustomization.ingress_tls

import rego.v1

# Ingress TLS Policy
# Ensures all Ingress resources have TLS configured

deny contains msg if {
    input.request.kind.kind == "Ingress"
    not has_tls_config(input.request.object)
    msg := sprintf("Ingress '%s' must have TLS configuration", [input.request.object.metadata.name])
}

# Check if ingress has at least one TLS host configured
has_tls_config(ingress) {
    tls := ingress.spec.tls
    count(tls) > 0
}

# Ensure TLS hosts match ingress rules
deny contains msg if {
    input.request.kind.kind == "Ingress"
    ingress := input.request.object
    tls_hosts := {host | host := ingress.spec.tls[_].hosts[_]}
    rule_hosts := {host | host := ingress.spec.rules[_].host}
    
    # Find hosts in rules but not in TLS
    missing_hosts := rule_hosts - tls_hosts
    count(missing_hosts) > 0
    
    msg := sprintf("Ingress '%s' has hosts without TLS coverage: %v", [ingress.metadata.name, missing_hosts])
}

# Ensure secretName is specified for each TLS entry
deny contains msg if {
    input.request.kind.kind == "Ingress"
    ingress := input.request.object
    some tls_entry in ingress.spec.tls
    not tls_entry.secretName
    
    msg := sprintf("Ingress '%s' has TLS entry without secretName specified", [ingress.metadata.name])
}

