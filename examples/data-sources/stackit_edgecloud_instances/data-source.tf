
# returns all Edge Cloud instances created in the given project which are inside the provider default_region
data "stackit_edgecloud_instances" "plan_id" {
  project_id = var.project_id
}

# returns all Edge Cloud instances created in the given project in the given region
data "stackit_edgecloud_instances" "plan_id" {
  project_id = var.project_id
  region     = var.region
}
