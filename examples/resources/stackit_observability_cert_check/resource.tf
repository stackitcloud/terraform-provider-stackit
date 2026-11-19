resource "stackit_observability_cert_check" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  source      = "tcp://www.example.com:443"
}
