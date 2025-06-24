variable "project_id" {}
variable "name" {}

provider "stackit" {
  default_region                     = "eu01"
  credentials_path                   = "~/.stackit/credentials.json"
  service_account_token              = ""
  service_account_key_path           = ""
  service_account_key                = ""
  private_key_path                   = ""
  private_key                        = ""
  service_account_email              = "abc@abc.de"
  argus_custom_endpoint              = "https://argus.api.eu01.stackit.cloud"
  cdn_custom_endpoint                = "https://cdn.api.eu01.stackit.cloud"
  dns_custom_endpoint                = "https://dns.api.stackit.cloud"
  git_custom_endpoint                = "https://git.api.stackit.cloud"
  iaas_custom_endpoint               = "https://iaas.api.stackit.cloud"
  mongodbflex_custom_endpoint        = "https://mongodbflex.api.stackit.cloud"
  modelserving_custom_endpoint       = "https://modelserving.api.stackit.cloud"
  loadbalancer_custom_endpoint       = "https://load-balancer.api.stackit.cloud"
  mariadb_custom_endpoint            = "https://mariadb.api.stackit.cloud"
  authorization_custom_endpoint      = "https://authorization.api.stackit.cloud"
  objectstorage_custom_endpoint      = "https://objectstorage.api.stackit.cloud"
  observability_custom_endpoint      = "https://observability.api.stackit.cloud"
  opensearch_custom_endpoint         = "https://opensearch.api.stackit.cloud"
  postgresflex_custom_endpoint       = "https://postgresflex.api.stackit.cloud"
  redis_custom_endpoint              = "https://redis.api.stackit.cloud"
  server_backup_custom_endpoint      = "https://server-backup.api.stackit.cloud"
  server_update_custom_endpoint      = "https://server-update.api.stackit.cloud"
  service_account_custom_endpoint    = "https://service-account.api.stackit.cloud"
  resourcemanager_custom_endpoint    = "https://resourcemanager.api.stackit.cloud"
  sqlserverflex_custom_endpoint      = "https://sqlserverflex.api.stackit.cloud"
  ske_custom_endpoint                = "https://ske.api.stackit.cloud"
  service_enablement_custom_endpoint = "https://service-enablement.api.stackit.cloud"
  token_custom_endpoint              = "https://token.api.stackit.cloud"
  enable_beta_resources              = "true"
}

resource "stackit_network" "network" {
  name       = var.name
  project_id = var.project_id
}
