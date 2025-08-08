resource "stackit_postgresflex_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-instance"
  acl             = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  backup_schedule = "00 00 * * *"
  flavor = {
    cpu = 2
    ram = 4
  }
  replicas = 3
  storage = {
    class = "class"
    size  = 5
  }
  version = 14
}

# Only use the import statement, if you want to import an existing postgresflex instance
import {
  to = stackit_postgresflex_instance.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id}"
}