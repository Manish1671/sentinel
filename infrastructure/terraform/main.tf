module "networking" {
  source               = "./modules/networking"
  project_name         = var.project_name
  environment          = var.environment
  vpc_cidr             = var.vpc_cidr
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
  availability_zones   = var.availability_zones
  cluster_name         = var.cluster_name
}

module "eks" {
  source             = "./modules/eks"
  project_name       = var.project_name
  environment        = var.environment
  cluster_name       = var.cluster_name
  vpc_id             = module.networking.vpc_id
  private_subnet_ids = module.networking.private_subnet_ids
  node_instance_type = var.node_instance_type
  node_desired_size  = var.node_desired_size
  node_min_size      = var.node_min_size
  node_max_size      = var.node_max_size
}

module "rds" {
  source                     = "./modules/rds"
  project_name               = var.project_name
  environment                = var.environment
  vpc_id                     = module.networking.vpc_id
  private_subnet_ids         = module.networking.private_subnet_ids
  allowed_security_group_ids = [module.eks.node_security_group_id]
  instance_class             = var.db_instance_class
  allocated_storage          = var.db_allocated_storage
  db_name                    = var.db_name
  username                   = var.db_username
}

module "redis" {
  source                     = "./modules/redis"
  project_name               = var.project_name
  environment                = var.environment
  vpc_id                     = module.networking.vpc_id
  private_subnet_ids         = module.networking.private_subnet_ids
  allowed_security_group_ids = [module.eks.node_security_group_id]
  node_type                  = var.redis_node_type
}

module "s3" {
  source       = "./modules/s3"
  project_name = var.project_name
  environment  = var.environment
}

module "iam" {
  source             = "./modules/iam"
  project_name       = var.project_name
  environment        = var.environment
  oidc_provider_arn  = module.eks.oidc_provider_arn
  oidc_provider_url  = module.eks.oidc_provider_url
  s3_bucket_arn      = module.s3.bucket_arn
  enable_github_oidc = var.enable_github_oidc
  github_org         = var.github_org
  github_repo        = var.github_repo
  aws_region         = var.aws_region
}
