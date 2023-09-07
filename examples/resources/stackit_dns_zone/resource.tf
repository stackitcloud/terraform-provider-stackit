resource "stackit_dns_zone" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name          = "Example zone"
  dns_name      = "www.example-zone.com"
  contact_email = "aa@bb.ccc"
  type          = "primary"
  acl           = "192.168.0.0/24"
  description   = "Example description"
  default_ttl   = 1230
}
