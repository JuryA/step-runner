module "vpc" {
  source      = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/vpc?ref=v0.13.1"
  metadata    = local.metadata
  zone        = var.aws_zone
  cidr        = "10.0.0.0/16"
  subnet_cidr = "10.0.0.0/24"
}

module "iam" {
  source   = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/iam?ref=v0.13.1"
  metadata = local.metadata
}

module "ami_lookup" {
  source   = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/ami_lookup?ref=v0.13.1"
  use_case = "aws-linux-ephemeral"
  region   = var.aws_region
  metadata = local.metadata
}

module "fleeting" {
  source   = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/fleeting?ref=v0.13.1"
  metadata = local.metadata
  service  = "ec2"
  os       = "linux"

  vpc = {
    enabled    = module.vpc.enabled
    id         = module.vpc.id
    subnet_ids = module.vpc.subnet_ids
  }

  security_group_ids = [module.security_groups.fleeting_id]
  instance_type        = local.ephemeral_runner.machine_type
  ephemeral_runner_ami = local.ephemeral_runner.source_image != "" ? local.ephemeral_runner.source_image : module.ami_lookup.ami_id
  scale_min            = var.autoscaling_policy.scale_min
  scale_max            = var.autoscaling_policy.scale_max
}

module "cache" {
  source   = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/cache?ref=v0.13.1"
  metadata = local.metadata
}

module "runner" {
  source   = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/runner?ref=v0.13.1"
  metadata = local.metadata

  vpc = {
    enabled    = module.vpc.enabled
    id         = module.vpc.id
    subnet_ids = module.vpc.subnet_ids
  }

  iam = {
    enabled                    = module.iam.enabled
    fleeting_access_key_id     = module.iam.fleeting_access_key_id
    fleeting_secret_access_key = module.iam.fleeting_secret_access_key
  }

  fleeting = {
    enabled                = module.fleeting.enabled
    ssh_key_pem_name       = module.fleeting.ssh_key_pem_name
    ssh_key_pem            = module.fleeting.ssh_key_pem
    autoscaling_group_name = module.fleeting.autoscaling_group_name
  }

  gitlab = {
    enabled      = module.gitlab.enabled
    runner_token = module.gitlab.runner_token
    url          = module.gitlab.url
  }

  cache = {
    enabled           = module.cache.enabled
    server_address    = module.cache.server_address
    bucket_name       = module.cache.bucket_name
    bucket_location   = module.cache.bucket_location
    access_key_id     = module.cache.access_key_id
    secret_access_key = module.cache.secret_access_key
  }

  service               = "ec2"
  executor              = "docker-autoscaler"
  scale_min             = var.autoscaling_policy.scale_min
  scale_max             = var.autoscaling_policy.scale_max
  idle_time             = var.autoscaling_policy.idle_time
  idle_percentage       = var.autoscaling_policy.idle_percentage
  capacity_per_instance = var.capacity_per_instance
  max_use_count         = var.max_use_count
  region                = var.aws_region
  runner_version        = var.runner_version

  security_group_ids = [module.security_groups.runner_manager_id]

  runners_global_section = <<EOS
  step_runner_image = "${var.step_runner_image}"
EOS
}

module "security_groups" {
  source   = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/aws/security_groups?ref=v0.13.1"
  metadata = local.metadata

  vpc = {
    enabled    = module.vpc.enabled
    id         = module.vpc.id
    subnet_ids = module.vpc.subnet_ids
  }
}

module "gitlab" {
  source             = "git::https://gitlab.com/gitlab-org/ci-cd/runner-tools/grit.git//modules/gitlab?ref=v0.13.1"
  metadata           = local.metadata
  url                = var.gitlab_base_url
  project_id         = var.gitlab_project_id
  runner_description = "Runner used in Steps E2E tests"
  runner_tags = ["steps-e2e", var.runner_tag]
}
