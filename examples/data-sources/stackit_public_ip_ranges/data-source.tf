data "stackit_public_ip_ranges" "example" {}

# example usage: allow stackit services and customer vpn cidr to access observability apis
locals {
  vpn_cidrs = ["X.X.X.X/32", "X.X.X.X/24"]
}

resource "stackit_observability_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  plan_name  = "Observability-Monitoring-Medium-EU01"
  # Allow all stackit services and customer vpn cidr to access observability apis
  acl                                    = concat(data.stackit_public_ip_ranges.example.cidr_list, local.vpn_cidrs)
  metrics_retention_days                 = 90
  metrics_retention_days_5m_downsampling = 90
  metrics_retention_days_1h_downsampling = 90
}