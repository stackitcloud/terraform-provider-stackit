variable "project_id" {}
variable "name" {}
variable "disk_format" {}
variable "local_file_path" {}
variable "min_disk_size" {}
variable "min_ram" {}
variable "label" {}
variable "boot_menu" {}
variable "cdrom_bus" {}
variable "disk_bus" {}
variable "nic_model" {}
variable "operating_system" {}
variable "operating_system_distro" {}
variable "operating_system_version" {}
variable "rescue_bus" {}
variable "rescue_device" {}
variable "secure_boot" {}
variable "uefi" {}
variable "video_model" {}
variable "virtio_scsi" {}

resource "stackit_image" "image" {
  project_id      = var.project_id
  name            = var.name
  disk_format     = var.disk_format
  local_file_path = var.local_file_path
  min_disk_size   = var.min_disk_size
  min_ram         = var.min_ram
  labels = {
    "acc-test" : var.label
  }
  config = {
    boot_menu                = var.boot_menu
    cdrom_bus                = var.cdrom_bus
    disk_bus                 = var.disk_bus
    nic_model                = var.nic_model
    operating_system         = var.operating_system
    operating_system_distro  = var.operating_system_distro
    operating_system_version = var.operating_system_version
    rescue_bus               = var.rescue_bus
    rescue_device            = var.rescue_device
    secure_boot              = var.secure_boot
    uefi                     = var.uefi
    video_model              = var.video_model
    virtio_scsi              = var.virtio_scsi
  }

}