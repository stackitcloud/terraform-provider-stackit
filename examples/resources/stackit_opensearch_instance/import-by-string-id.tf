# Only use the import statement, if you want to import an existing opensearch instance
import {
  to = stackit_opensearch_instance.import-example
  id = "${var.project_id},${var.instance_id}"
}
