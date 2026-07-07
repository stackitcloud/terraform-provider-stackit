# Only use the import statement, if you want to import an existing observability logalertgroup
import {
  to = stackit_observability_logalertgroup.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_logalertgroup_name}"
}
