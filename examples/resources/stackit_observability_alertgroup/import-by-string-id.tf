# Only use the import statement, if you want to import an existing observability alertgroup
import {
  to = stackit_observability_alertgroup.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_alertgroup_name}"
}
