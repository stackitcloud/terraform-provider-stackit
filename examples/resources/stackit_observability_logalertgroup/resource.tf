resource "stackit_observability_logalertgroup" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example-log-alert-group"
  interval    = "60m"
  rules = [
    {
      alert      = "example-log-alert-name"
      expression = "sum(rate({namespace=\"example\", pod=\"logger\"} |= \"Simulated error message\" [1m])) > 0"
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
      alert      = "example-log-alert-name-2"
      expression = "sum(rate({namespace=\"example\", pod=\"logger\"} |= \"Another error message\" [1m])) > 0"
      for        = "60s"
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

# Only use the import statement, if you want to import an existing observability logalertgroup
import {
  to = stackit_observability_logalertgroup.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_logalertgroup_name}"
}
