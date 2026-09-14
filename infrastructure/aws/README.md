# infrastructure/aws

AWS-specific notes and environment conventions.

Planned production landing zone (later):

- EKS for services
- RDS PostgreSQL
- ElastiCache Redis
- MSK or equivalent Kafka
- S3 for artifacts
- IAM roles for service identities (no long-lived keys in app config when possible)

This folder is documentation-first until Terraform modules exist.
