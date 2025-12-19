# Copyright (c) STACKIT

resource "stackitprivatepreview_postgresflexalpha_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username    = "username"
  roles       = ["role"]
}

# Only use the import statement, if you want to import an existing postgresflex user
import {
  to = stackitprivatepreview_postgresflexalpha_user.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id},${var.user_id}"
}