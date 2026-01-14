# the instance resource only exists here to illustrate the usage of it's attribute
resource "stackit_edgecloud_instance" "this" {
  project_id   = local.project_id
  display_name = "example"
  plan_id      = var.plan_id
  description  = "some_description"
}

resource "stackit_edgecloud_kubeconfig" "by_name" {
  project_id    = var.project_id
  instance_name = stackit_edgecloud_instance.this.display_name
  expiration    = 3600 # seconds
}

resource "stackit_edgecloud_kubeconfig" "by_id" {
  project_id  = var.project_id
  instance_id = stackit_edgecloud_instance.this.instance_id
  expiration  = 3600 # seconds
}