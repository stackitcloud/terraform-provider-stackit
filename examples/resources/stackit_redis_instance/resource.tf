resource "stackit_redis_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  version    = "7"
  plan_name  = "stackit-redis-1.2.10-replica"
  parameters = {
    sgw_acl                 = "193.148.160.0/19,45.129.40.0/21,45.135.244.0/22"
    enable_monitoring       = false
    down_after_milliseconds = 30000
    syslog                  = ["logs4.your-syslog-endpoint.com:54321"]
  }
}
