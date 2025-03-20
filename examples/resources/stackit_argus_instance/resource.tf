resource "stackit_argus_instance" "example" {
  project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                                   = "example-instance"
  plan_name                              = "Monitoring-Medium-EU01"
  acl                                    = ["1.1.1.1/32", "2.2.2.2/32"]
  metrics_retention_days                 = 7
  metrics_retention_days_5m_downsampling = 30
  metrics_retention_days_1h_downsampling = 365
}

### Example move

#	Example to move the deprecated `stackit_argus_instance` resource to the new`stackit_observability_instance` resource:
#	1. Add a new `stackit_observability_instance` resource with the same values like your previous `stackit_argus_instance` resource.
#	2. Add a moved block which reference the `stackit_argus_instance` and `stackit_observability_instance` resource.
#	3. Remove your old `stackit_argus_instance` resource and run `$ terraform apply`.

resource "stackit_argus_instance" "example" {
  project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                                   = "example-instance"
  plan_name                              = "Monitoring-Medium-EU01"
  acl                                    = ["1.1.1.1/32", "2.2.2.2/32"]
  metrics_retention_days                 = 7
  metrics_retention_days_5m_downsampling = 30
  metrics_retention_days_1h_downsampling = 365
}

moved {
  from = stackit_argus_instance.example
  to   = stackit_observability_instance.example
}

resource "stackit_observability_instance" "example" {
  project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                                   = "example-instance"
  plan_name                              = "Monitoring-Medium-EU01"
  acl                                    = ["1.1.1.1/32", "2.2.2.2/32"]
  metrics_retention_days                 = 7
  metrics_retention_days_5m_downsampling = 30
  metrics_retention_days_1h_downsampling = 365
}
