resource "stackit_sqlserverflex_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-instance"
  flavor_id       = "4.16-Single"
  backup_schedule = "0 0 * * *"
  network = {
    acl = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  }
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  retention_days = 32
  version        = "2022"
}
