resource "stackit_rabbitmq_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  version    = "3.13"
  plan_name  = "stackit-rabbitmq-1.2.10-replica"
  parameters = {
    sgw_acl           = "193.148.160.0/19,45.129.40.0/21,45.135.244.0/22"
    consumer_timeout  = 18000000
    enable_monitoring = false
    plugins           = ["rabbitmq_consistent_hash_exchange", "rabbitmq_federation", "rabbitmq_tracing"]
  }
}
