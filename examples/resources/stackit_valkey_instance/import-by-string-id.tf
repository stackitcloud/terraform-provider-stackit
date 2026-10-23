# Only use the import statement, if you want to import an existing valkey instance
import {
  to = stackit_valkey_instance.import-example
  id = "${var.project_id},${var.region},${var.valkey_instance_id}"
}
