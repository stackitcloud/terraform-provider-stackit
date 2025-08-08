resource "stackit_observability_alertgroup" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example-alert-group"
  interval    = "60s"
  rules = [
    {
      alert      = "example-alert-name"
      expression = "kube_node_status_condition{condition=\"Ready\", status=\"false\"} > 0"
      for        = "60s"
      labels = {
        severity = "critical"
      },
      annotations = {
        summary : "example summary"
        description : "example description"
      }
    },
    {
      alert      = "example-alert-name-2"
      expression = "kube_node_status_condition{condition=\"Ready\", status=\"false\"} > 0"
      for        = "1m"
      labels = {
        severity = "critical"
      },
      annotations = {
        summary : "example summary"
        description : "example description"
      }
    },
  ]
}

# Only use the import statement, if you want to import an existing observability alertgroup
import {
  to = stackit_observability_alertgroup.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_alertgroup_name}"
}
