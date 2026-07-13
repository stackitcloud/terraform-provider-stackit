# Only use the import statement, if you want to import an existing rabbitmq credential
import {
  to = stackit_rabbitmq_credential.import-example
  id = "${var.project_id},${var.rabbitmq_instance_id},${var.rabbitmq_credential_id}"
}
