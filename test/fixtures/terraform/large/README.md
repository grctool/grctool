# Large Terraform Fixtures

This directory contains large-scale Terraform configurations for benchmark testing.

**Size:** ~200-250 resources across ~40-50 files
**Purpose:** Enterprise multi-account infrastructure
**Coverage:** Multi-region VPC, EKS, RDS, S3, KMS, IAM, CloudWatch, CloudTrail, Backup, DR

## Structure

- `network_*.tf` - Multi-region VPC configurations
- `compute_*.tf` - Multiple EKS clusters across regions
- `database_*.tf` - RDS clusters (prod, staging, dev, dr)
- `storage_*.tf` - Numerous S3 buckets with encryption
- `security_*.tf` - KMS keys, IAM roles, policies
- `monitoring_*.tf` - CloudWatch, CloudTrail, Config
- `backup_*.tf` - AWS Backup configurations
- `dr_*.tf` - Disaster recovery resources
