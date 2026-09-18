variable "project_name" { type = string }
variable "environment" { type = string }

data "aws_caller_identity" "current" {}

resource "aws_s3_bucket" "artifacts" {
  bucket = "${var.project_name}-${var.environment}-artifacts-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket_public_access_block" "artifacts" {
  bucket                  = aws_s3_bucket.artifacts.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  versioning_configuration {
    status = "Enabled"
  }
}

output "bucket_id" { value = aws_s3_bucket.artifacts.id }
output "bucket_arn" { value = aws_s3_bucket.artifacts.arn }
