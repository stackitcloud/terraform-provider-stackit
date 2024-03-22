resource "stackit_argus_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  plan_name  = "Monitoring-Medium-EU01"
  acl        = ["1.1.1.1/32", "2.2.2.2/32"]
}
