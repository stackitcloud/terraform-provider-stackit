# Only use the import statement, if you want to import an existing Edge Cloud instance resource
import {
  to = stackit_edgecloud_instance.this
  id = "${local.project_id},${local.region},INSTANCE_ID"
}
