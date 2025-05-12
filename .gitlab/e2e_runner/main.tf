terraform {
  required_providers {
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = ">=17.0.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "http" {}
}

locals {
  aws_zone        = "us-east-1a"
  aws_region      = "us-east-1"
  gitlab_base_url = "https://gitlab.com"
  runner_version  = "17.11.1-1"
}

provider "gitlab" {
  base_url = local.gitlab_base_url
}

provider "aws" {
  region = local.aws_region
}

module "deploy_e2e_runner" {
  source                = "./deploy"
  aws_zone              = local.aws_zone
  aws_region            = local.aws_region
  gitlab_base_url       = local.gitlab_base_url
  gitlab_project_id     = var.gitlab_project_id
  runner_tag            = var.runner_tag
  runner_version        = local.runner_version
  step_runner_image     = var.step_runner_image
  capacity_per_instance = 1
  max_use_count         = 10

  autoscaling_policy = {
    scale_min       = 1    # affects idle_count, which is scale_min * capacity_per_instance
    scale_max       = 5    # affects concurrent, which is scale_max * capacity_per_instance
    idle_time       = "5m" # how long idle capacity is kept around for, useful when idle_count decreases
    idle_percentage = 1.5  # idle_count = max(active tasks * idle_percentage, idle_count), starts instances to take on future tasks
  }
}
