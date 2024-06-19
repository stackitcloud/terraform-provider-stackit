resource "stackit_rabbitmq_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  version    = "10"
  plan_name  = "example-plan-name"
  parameters = {
    sgw_acl           = "x.x.x.x/x,y.y.y.y/y"
    consumer_timeout  = 18000000
    enable_monitoring = false
    plugins           = ["example-plugin1", "example-plugin2"]
  }
}
