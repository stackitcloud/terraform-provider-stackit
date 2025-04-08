resource "stackit_observability_instance" "example" {
  project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                                   = "example-instance"
  plan_name                              = "Observability-Monitoring-Medium-EU01"
  acl                                    = ["1.1.1.1/32", "2.2.2.2/32"]
  metrics_retention_days                 = 7
  metrics_retention_days_5m_downsampling = 30
  metrics_retention_days_1h_downsampling = 365
}
