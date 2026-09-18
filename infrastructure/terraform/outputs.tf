output "vpc_id" {
  value = module.networking.vpc_id
}

output "private_subnet_ids" {
  value = module.networking.private_subnet_ids
}

output "public_subnet_ids" {
  value = module.networking.public_subnet_ids
}

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  value = module.rds.endpoint
}

output "rds_secret_arn" {
  value = module.rds.secret_arn
}

output "redis_endpoint" {
  value = module.redis.primary_endpoint
}

output "s3_bucket_id" {
  value = module.s3.bucket_id
}

output "workload_role_arn" {
  value = module.iam.workload_role_arn
}

output "github_actions_role_arn" {
  value = module.iam.github_role_arn
}

output "kafka_note" {
  value = "Kafka is not provisioned. Set KAFKA_BROKERS to an operator-managed cluster. MSK is not included."
}
