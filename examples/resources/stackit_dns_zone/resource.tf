resource "stackit_dns_zone" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name          = "Example zone"
  dns_name      = "example-zone.com"
  contact_email = "aa@bb.ccc"
  type          = "primary"
  acl           = "192.168.0.0/24"
  description   = "Example description"
  default_ttl   = 1230
}

# Only use the import statement, if you want to import an existing dns zone
import {
  to = stackit_dns_zone.import-example
  id = "${var.project_id},${var.zone_id}"
}