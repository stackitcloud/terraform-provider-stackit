resource "stackit_mariadb_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  version    = "10.11"
  plan_name  = "stackit-mariadb-1.2.10-replica"
  parameters = {
    sgw_acl = "193.148.160.0/19,45.129.40.0/21,45.135.244.0/22"
  }
}

# Only use the import statement, if you want to import an existing mariadb instance
import {
  to = stackit_mariadb_instance.import-example
  id = "${var.project_id},${var.mariadb_instance_id}"
}
