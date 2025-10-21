package main

import rego.v1

budget_component_label := "github.com/nvatuan/domains"

deny contains msg if {
  input.kind == "Deployment"
  not input.spec.template.metadata.labels[budget_component_label]
  msg = sprintf("Deployment %s does not have the required label '%s'", [input.metadata.name, budget_component_label])
}

deny contains msg if {
  input.kind == "StatefulSet"
  not input.spec.template.metadata.labels[budget_component_label]
  msg = sprintf("StatefulSet %s does not have the required label '%s'", [input.metadata.name, budget_component_label])
}

deny contains msg if {
  input.kind == "Job"
  not input.spec.template.metadata.labels[budget_component_label]
  msg = sprintf("Job %s does not have the required label '%s'", [input.metadata.name, budget_component_label])
}

deny contains msg if {
  input.kind == "CronJob"
  not input.spec.jobTemplate.spec.template.metadata.labels[budget_component_label]
  msg = sprintf("CronJob %s does not have the required label '%s'", [input.metadata.name, budget_component_label])
}