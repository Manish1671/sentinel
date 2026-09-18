variable "project_name" { type = string }
variable "environment" { type = string }
variable "vpc_id" { type = string }
variable "private_subnet_ids" { type = list(string) }
variable "allowed_security_group_ids" { type = list(string) }
variable "node_type" { type = string }

resource "aws_elasticache_subnet_group" "this" {
  name       = "${var.project_name}-${var.environment}-redis"
  subnet_ids = var.private_subnet_ids
}

resource "aws_security_group" "this" {
  name        = "${var.project_name}-${var.environment}-redis"
  description = "Redis from EKS nodes"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Redis from EKS"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = var.allowed_security_group_ids
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_elasticache_cluster" "this" {
  cluster_id           = "${var.project_name}-${var.environment}"
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = var.node_type
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"
  port                 = 6379
  subnet_group_name    = aws_elasticache_subnet_group.this.name
  security_group_ids   = [aws_security_group.this.id]
}

output "primary_endpoint" { value = aws_elasticache_cluster.this.cache_nodes[0].address }
output "port" { value = aws_elasticache_cluster.this.port }
output "security_group_id" { value = aws_security_group.this.id }
