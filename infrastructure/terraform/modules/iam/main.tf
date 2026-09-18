variable "project_name" { type = string }
variable "environment" { type = string }
variable "oidc_provider_arn" { type = string }
variable "oidc_provider_url" { type = string }
variable "s3_bucket_arn" { type = string }
variable "enable_github_oidc" { type = bool }
variable "github_org" { type = string }
variable "github_repo" { type = string }
variable "aws_region" { type = string }

locals {
  oidc_host = replace(var.oidc_provider_url, "https://", "")
}

data "aws_iam_policy_document" "workload_assume" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [var.oidc_provider_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "${local.oidc_host}:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "${local.oidc_host}:sub"
      values = [
        "system:serviceaccount:sentinel:sentinel-api",
        "system:serviceaccount:sentinel:sentinel-ai"
      ]
    }
  }
}

resource "aws_iam_role" "workload" {
  name               = "${var.project_name}-${var.environment}-workload"
  assume_role_policy = data.aws_iam_policy_document.workload_assume.json
}

data "aws_iam_policy_document" "workload" {
  statement {
    sid = "S3Artifacts"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket"
    ]
    resources = [
      var.s3_bucket_arn,
      "${var.s3_bucket_arn}/*"
    ]
  }
}

resource "aws_iam_policy" "workload" {
  name   = "${var.project_name}-${var.environment}-workload"
  policy = data.aws_iam_policy_document.workload.json
}

resource "aws_iam_role_policy_attachment" "workload" {
  role       = aws_iam_role.workload.name
  policy_arn = aws_iam_policy.workload.arn
}

data "aws_iam_policy_document" "github_assume" {
  count = var.enable_github_oidc ? 1 : 0
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github[0].arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_org}/${var.github_repo}:*"]
    }
  }
}

resource "aws_iam_openid_connect_provider" "github" {
  count           = var.enable_github_oidc ? 1 : 0
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["ffffffffffffffffffffffffffffffffffffffff"]
}

resource "aws_iam_role" "github" {
  count              = var.enable_github_oidc ? 1 : 0
  name               = "${var.project_name}-${var.environment}-github-actions"
  assume_role_policy = data.aws_iam_policy_document.github_assume[0].json
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "github" {
  count = var.enable_github_oidc ? 1 : 0
  statement {
    sid       = "ECRAuth"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }
  statement {
    sid = "ECRPush"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage",
      "ecr:InitiateLayerUpload",
      "ecr:UploadLayerPart",
      "ecr:CompleteLayerUpload",
      "ecr:PutImage"
    ]
    resources = [
      "arn:aws:ecr:${var.aws_region}:${data.aws_caller_identity.current.account_id}:repository/${var.project_name}-*"
    ]
  }
  statement {
    sid       = "EKSRead"
    actions   = ["eks:DescribeCluster", "eks:ListClusters"]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "github" {
  count  = var.enable_github_oidc ? 1 : 0
  name   = "${var.project_name}-${var.environment}-github-actions"
  policy = data.aws_iam_policy_document.github[0].json
}

resource "aws_iam_role_policy_attachment" "github" {
  count      = var.enable_github_oidc ? 1 : 0
  role       = aws_iam_role.github[0].name
  policy_arn = aws_iam_policy.github[0].arn
}

output "workload_role_arn" { value = aws_iam_role.workload.arn }
output "github_role_arn" { value = try(aws_iam_role.github[0].arn, "") }
