resource "stackit_ske_kubeconfig" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  cluster_name = "example-cluster"

  refresh        = true
  expiration     = 7200 # 2 hours
  refresh_before = 3600 # 1 hour
}
