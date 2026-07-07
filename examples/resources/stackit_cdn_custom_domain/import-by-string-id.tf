# Only use the import statement, if you want to import an existing cdn custom domain
import {
  to = stackit_cdn_custom_domain.import-example
  id = "${var.project_id},${var.distribution_id},${var.custom_domain_name}"
}
