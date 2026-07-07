# Only use the import statement, if you want to import an existing redis instance
import {
  to = stackit_redis_instance.import-example
  id = "${var.project_id},${var.redis_instance_id}"
}
