# Only use the import statement, if you want to import an existing mariadb instance
import {
  to = stackit_mariadb_instance.import-example
  id = "${var.project_id},${var.mariadb_instance_id}"
}
