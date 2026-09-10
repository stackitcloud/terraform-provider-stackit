resource "stackit_ufw_instance" "rule" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region      = "eu01"
  instance_id = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
  product     = "edge-cloud"
  source_ip   = "1.2.3.4/32"
  type        = "ACL"
}