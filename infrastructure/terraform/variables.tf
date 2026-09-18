variable "aws_region" {
  type        = string
  description = "AWS region"
  default     = "us-east-1"
}

variable "environment" {
  type        = string
  description = "Environment name (demo, staging, prod)"
  default     = "demo"
}

variable "project_name" {
  type        = string
  description = "Project name used in resource names"
  default     = "sentinel"
}

variable "cluster_name" {
  type        = string
  description = "EKS cluster name"
  default     = "sentinel-demo"
}

variable "vpc_cidr" {
  type    = string
  default = "10.40.0.0/16"
}

variable "public_subnet_cidrs" {
  type    = list(string)
  default = ["10.40.0.0/24", "10.40.1.0/24"]
}

variable "private_subnet_cidrs" {
  type    = list(string)
  default = ["10.40.10.0/24", "10.40.11.0/24"]
}

variable "availability_zones" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b"]
}

variable "node_instance_type" {
  type        = string
  description = "EKS managed node instance type (starting size, not benchmarked)"
  default     = "t3.medium"
}

variable "node_desired_size" {
  type    = number
  default = 2
}

variable "node_min_size" {
  type    = number
  default = 1
}

variable "node_max_size" {
  type    = number
  default = 3
}

variable "db_instance_class" {
  type        = string
  description = "RDS instance class (starting size, not benchmarked)"
  default     = "db.t4g.micro"
}

variable "db_allocated_storage" {
  type    = number
  default = 20
}

variable "db_name" {
  type    = string
  default = "sentinel"
}

variable "db_username" {
  type    = string
  default = "sentinel"
}

variable "redis_node_type" {
  type        = string
  description = "ElastiCache node type (starting size, not benchmarked)"
  default     = "cache.t4g.micro"
}

variable "enable_github_oidc" {
  type        = bool
  description = "Create GitHub Actions OIDC provider and deploy role"
  default     = false
}

variable "github_org" {
  type    = string
  default = "sentinel-dev"
}

variable "github_repo" {
  type    = string
  default = "sentinel"
}

variable "tags" {
  type    = map(string)
  default = {}
}
