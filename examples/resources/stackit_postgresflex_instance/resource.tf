resource "stackit_postgresflex_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  network = {
    acl = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  }
  backup_schedule = "0 0 * * *"
  flavor_id       = "4.8-replica"
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  version = "14"
  retention_days = 32
}
