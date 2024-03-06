resource "stackit_ske_kubeconfig" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  cluster_name = "example-cluster"
  refresh      = true
}
