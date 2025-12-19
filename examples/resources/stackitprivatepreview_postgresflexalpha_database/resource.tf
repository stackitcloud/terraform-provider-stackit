# Copyright (c) STACKIT

resource "stackitprivatepreview_postgresflexalpha_database" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "mydb"
  owner       = "myusername"
}

# Only use the import statement, if you want to import an existing postgresflex database
import {
  to = stackitprivatepreview_postgresflexalpha_database.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id},${var.postgres_database_id}"
}