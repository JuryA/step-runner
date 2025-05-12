locals {
  metadata = {
    name = var.runner_tag
    labels = {
      managed = "grit"
    }
    min_support = "experimental"
  }

  ephemeral_runner = {
    disk_type    = "gp3"
    disk_size    = 25
    machine_type = "t3.medium"
    source_image = ""
  }
}
