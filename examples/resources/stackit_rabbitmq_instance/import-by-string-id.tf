# Only use the import statement, if you want to import an existing rabbitmq instance
import {
  to = stackit_rabbitmq_instance.import-example
  id = "${var.project_id},${var.rabbitmq_instance_id}"
}
