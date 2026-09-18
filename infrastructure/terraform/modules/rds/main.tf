variable "project_name" { type = string }
variable "environment" { type = string }
variable "vpc_id" { type = string }
variable "private_subnet_ids" { type = list(string) }
variable "allowed_security_group_ids" { type = list(string) }
variable "instance_class" { type = string }
variable "allocated_storage" { type = number }
variable "db_name" { type = string }
variable "username" { type = string }

resource "random_password" "master" {
  length  = 24
  special = false
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.project_name}-${var.environment}-rds"
  subnet_ids = var.private_subnet_ids
}

resource "aws_security_group" "this" {
  name        = "${var.project_name}-${var.environment}-rds"
  description = "PostgreSQL from EKS nodes"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Postgres from EKS"
    from_port       = 5432
    to_port         = 5432
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

resource "aws_db_instance" "this" {
  identifier                   = "${var.project_name}-${var.environment}"
  engine                       = "postgres"
  engine_version               = "16.4"
  instance_class               = var.instance_class
  allocated_storage            = var.allocated_storage
  db_name                      = var.db_name
  username                     = var.username
  password                     = random_password.master.result
  db_subnet_group_name         = aws_db_subnet_group.this.name
  vpc_security_group_ids       = [aws_security_group.this.id]
  storage_encrypted            = true
  skip_final_snapshot          = true
  publicly_accessible          = false
  backup_retention_period      = 7
  deletion_protection          = false
  performance_insights_enabled = false
}

resource "aws_secretsmanager_secret" "db" {
  name = "${var.project_name}/${var.environment}/rds"
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id
  secret_string = jsonencode({
    username = var.username
    password = random_password.master.result
    host     = aws_db_instance.this.address
    port     = aws_db_instance.this.port
    dbname   = var.db_name
  })
}

output "endpoint" { value = aws_db_instance.this.address }
output "port" { value = aws_db_instance.this.port }
output "secret_arn" { value = aws_secretsmanager_secret.db.arn }
output "security_group_id" { value = aws_security_group.this.id }
