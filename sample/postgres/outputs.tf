# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

output "postgres_flavor" {
  value = data.stackitprivatepreview_postgresflexalpha_flavor.pgsql_flavor.flavor_id
}
