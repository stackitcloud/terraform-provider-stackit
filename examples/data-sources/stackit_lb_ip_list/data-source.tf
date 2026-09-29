data "stackit_lb_ip_list" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region     = "eu01"
  name       = "example-ip-list"
}
