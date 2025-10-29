resource "stackit_observability_instance" "example" {
  project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                                   = "example-instance"
  plan_name                              = "Observability-Starter-EU01"
  acl                                    = ["1.1.1.1/32", "2.2.2.2/32"]
  logs_retention_days                    = 30
  traces_retention_days                  = 30
  metrics_retention_days                 = 90
  metrics_retention_days_5m_downsampling = 90
  metrics_retention_days_1h_downsampling = 90
}

# Only use the import statement, if you want to import an existing observability instance
import {
  to = stackit_observability_instance.import-example
  id = "${var.project_id},${var.observability_instance_id}"
}