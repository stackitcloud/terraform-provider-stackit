resource "stackit_secretsmanager_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  acl        = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
}
