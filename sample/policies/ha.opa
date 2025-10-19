package kustomization.ha

import rego.v1

# High Availability Policy
# Ensures deployments have minimum replicas and proper node anti-affinity

# Check if deployment has minimum 2 replicas
deny contains msg if {
    input.request.kind.kind == "Deployment"
    replicas := input.request.object.spec.replicas
    replicas < 2
    msg := sprintf("Deployment '%s' must have at least 2 replicas for high availability, found: %d", [input.request.object.metadata.name, replicas])
}

# Check if deployment has PodAntiAffinity for node distribution
deny contains msg if {
    input.request.kind.kind == "Deployment"
    not has_node_anti_affinity(input.request.object)
    msg := sprintf("Deployment '%s' must have PodAntiAffinity with topologyKey 'kubernetes.io/hostname' for high availability", [input.request.object.metadata.name])
}

# Helper function to check if deployment has proper node anti-affinity
has_node_anti_affinity(deployment) if {
    affinity := deployment.spec.template.spec.affinity
    pod_anti_affinity := affinity.podAntiAffinity
    required_during_scheduling := pod_anti_affinity.requiredDuringSchedulingIgnoredDuringExecution

    some term in required_during_scheduling
    term.topologyKey == "kubernetes.io/hostname"
    some label_selector in term.labelSelector.matchExpressions
    label_selector.key == "app"
    label_selector.operator == "In"
    label_selector.values[_] == deployment.metadata.labels.app
}
