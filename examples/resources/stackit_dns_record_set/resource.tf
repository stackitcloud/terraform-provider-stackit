resource "stackit_dns_record_set" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  zone_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-record-set"
  type       = "A"
  comment    = "Example comment"
  records    = ["1.2.3.4"]
}
