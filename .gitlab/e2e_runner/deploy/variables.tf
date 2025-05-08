variable "gitlab_project_id" {
  type = string
}

variable "runner_tag" {
  type = string
}

variable "step_runner_image" {
  type = string
}

variable "runner_version" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "aws_zone" {
  type = string
}

variable "gitlab_base_url" {
  type = string
}

variable "capacity_per_instance" {
  type = number
}

variable "max_use_count" {
  type = number
}

variable "autoscaling_policy" {
  type = object({
    scale_min = optional(number, 1)
    scale_max = optional(number, 3)
    idle_time = optional(string, "5m")
    idle_percentage = optional(number, 1)
  })

  default = {
    scale_min       = 1
    scale_max       = 3
    idle_time       = "5m"
    idle_percentage = 1
  }
}
