resource "stackitprivatepreview_sqlserverflexalpha_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-instance"
  acl             = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  backup_schedule = "00 00 * * *"
  flavor = {
    cpu = 4
    ram = 16
  }
  storage = {
    class = "class"
    size  = 5
  }
  version = 2022
}

# Only use the import statement, if you want to import an existing sqlserverflex instance
import {
  to = stackitprivatepreview_sqlserverflexalpha_instance.import-example
  id = "${var.project_id},${var.region},${var.sql_instance_id}"
}
