
variable "project_id" {}
variable "name" {}
variable "acl" {}
variable "flavor_cpu" {}
variable "flavor_ram" {}
variable "replicas" {}
variable "storage_class" {}
variable "storage_size" {}
variable "version_db" {}
variable "options_type" {}
variable "flavor_id" {}
variable "backup_schedule" {}
variable "backup_schedule_read" {}

variable "snapshot_retention_days" {}
variable "daily_snapshot_retention_days" {}
variable "weekly_snapshot_retention_weeks" {}
variable "monthly_snapshot_retention_months" 
variable "point_in_time_window_hours" {}

variable "username" {}
variable "role" {}
variable "database" {}


resource "stackit_mongodbflex_instance" "instance" {
    project_id = var.project_id
    name    = var.name
    acl = [var.acl]
    flavor = {
        cpu = var.flavor_cpu
        ram = var.flavor_ram
    }
    replicas = var.replicas
    storage = {
        class = var.storage_class
        size = var.storage_size
    }
    version = var.version_db
    options = {
        type = var.options_type
        snapshot_retention_days = var.snapshot_retention_days
		daily_snapshot_retention_days = var.daily_snapshot_retention_days
        weekly_snapshot_retention_weeks = var.weekly_snapshot_retention_weeks
	    monthly_snapshot_retention_months = var.monthly_snapshot_retention_months
	    point_in_time_window_hours = var.point_in_time_window_hours
    }
    backup_schedule = var.backup_schedule
}

resource "stackit_mongodbflex_user" "user" {
    project_id = stackit_mongodbflex_instance.instance.project_id
    instance_id = stackit_mongodbflex_instance.instance.instance_id
    username = var.username
    roles = [var.role]
    database = var.database
}
