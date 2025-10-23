# Medium Terraform Fixtures

This directory contains medium-sized Terraform configurations for benchmark testing.

**Size:** ~50-75 resources across ~15-20 files
**Purpose:** Realistic multi-tier application infrastructure
**Coverage:** VPC, EKS, RDS, S3, KMS, IAM, CloudWatch, CloudTrail, Backup

## Structure

- `network.tf` - VPC, subnets, routing
- `compute_*.tf` - EKS cluster and node groups
- `database_*.tf` - RDS clusters (prod, staging)
- `storage_*.tf` - S3 buckets with encryption
- `security_*.tf` - KMS keys, IAM roles
- `monitoring_*.tf` - CloudWatch, CloudTrail
- `backup_*.tf` - AWS Backup configurations
