resource "stackit_sqlserverflex_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-instance"
  acl             = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  backup_schedule = "00 00 * * *"
  flavor_id       = "4.16-Single"
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  version = 2022
}
