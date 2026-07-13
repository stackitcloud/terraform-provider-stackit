# Only use the import statement, if you want to import an existing mongodbflex instance
import {
  to = stackit_mongodbflex_instance.import-example
  id = "${var.project_id},${var.region},${var.instance_id}"
}
