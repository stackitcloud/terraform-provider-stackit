# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

data "stackitprivatepreview_sqlserverflexalpha_flavor" "sqlserver_flavor" {
  project_id    = var.project_id
  region        = "eu01"
  cpu           = 4
  ram           = 16
  node_type     = "Single"
  storage_class = "premium-perf2-stackit"
}

output "sqlserver_flavor" {
  value = data.stackitprivatepreview_sqlserverflexalpha_flavor.sqlserver_flavor.flavor_id
}
