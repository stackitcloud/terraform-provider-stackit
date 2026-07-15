resource "stackit_vpc_region" "region" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  vpc_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_vpc_network_range" "example" {
  # Network range can only be created after the VPC is enabled for the region
  depends_on = [stackit_vpc_region.region]

  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  vpc_id      = stackit_vpc_region.region.vpc_id
  ip_version  = "ipv4"
  prefix      = "192.168.0.0/24"
  description = "my vpc network range"
}
