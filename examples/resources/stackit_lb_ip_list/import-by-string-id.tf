# Only use the import statement, if you want to import an existing load balancer IP list resource
import {
  to = stackit_loadbalancer_ip_list.import-example
  id = "${var.project_id},${var.region},${var.name}"
}
