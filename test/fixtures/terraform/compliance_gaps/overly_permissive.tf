# Compliance Gaps: Overly Permissive Security Configurations
# Purpose: Test fixtures for detecting overly permissive network and IAM configurations
# Expected Findings: HIGH severity - security groups with 0.0.0.0/0, wildcard IAM policies

# Security group allowing SSH from anywhere (COMPLIANCE GAP - HIGH)
resource "aws_security_group" "ssh_open" {
  name        = "ssh-from-anywhere"
  description = "Overly permissive SSH access"
  vpc_id      = "vpc-abc123"

  # SSH from 0.0.0.0/0 is a major security risk
  ingress {
    description = "SSH from anywhere"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # HIGH SEVERITY FINDING
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name  = "ssh-open"
    Issue = "overly-permissive-ingress"
  }
}

# Security group allowing all traffic from anywhere (COMPLIANCE GAP - HIGH)
resource "aws_security_group" "all_open" {
  name        = "all-traffic-open"
  description = "Allows all traffic from anywhere"
  vpc_id      = "vpc-abc123"

  # All ports, all protocols from anywhere
  ingress {
    description = "All traffic from anywhere"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"] # CRITICAL FINDING
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name  = "all-open"
    Issue = "critically-permissive"
  }
}

# Security group allowing database access from anywhere (COMPLIANCE GAP - HIGH)
resource "aws_security_group" "database_open" {
  name        = "database-from-anywhere"
  description = "Database accessible from internet"
  vpc_id      = "vpc-abc123"

  # PostgreSQL from anywhere
  ingress {
    description = "PostgreSQL from anywhere"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # HIGH SEVERITY - database should never be public
  }

  # MySQL from anywhere
  ingress {
    description = "MySQL from anywhere"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # HIGH SEVERITY
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name  = "database-open"
    Issue = "database-exposed-to-internet"
  }
}

# IAM policy with wildcard actions and resources (COMPLIANCE GAP - MEDIUM/HIGH)
resource "aws_iam_policy" "admin_wildcard" {
  name        = "overly-permissive-policy"
  description = "Policy with wildcard permissions"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "*"           # FINDING: Wildcard action
        Resource = "*"         # FINDING: Wildcard resource
      }
    ]
  })

  tags = {
    Name  = "admin-wildcard"
    Issue = "wildcard-permissions"
  }
}

# IAM policy allowing S3 full access on all buckets (COMPLIANCE GAP - MEDIUM)
resource "aws_iam_policy" "s3_wildcard" {
  name        = "s3-full-access-all-buckets"
  description = "Full S3 access on all buckets"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:*" # Wildcard S3 actions
        ]
        Resource = "*" # All buckets
      }
    ]
  })

  tags = {
    Name  = "s3-wildcard"
    Issue = "overly-broad-s3-access"
  }
}

# IAM role with overly broad assume role policy (COMPLIANCE GAP - MEDIUM)
resource "aws_iam_role" "assume_anyone" {
  name        = "assumable-by-anyone"
  description = "Role that can be assumed by any AWS account"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = "*" # Any AWS account can assume this role - FINDING
        }
      }
    ]
  })

  tags = {
    Name  = "assume-anyone"
    Issue = "overly-broad-assume-role"
  }
}

# S3 bucket with public read access (COMPLIANCE GAP - HIGH)
resource "aws_s3_bucket" "public_bucket" {
  bucket = "public-readable-bucket"

  tags = {
    Name  = "public-bucket"
    Issue = "public-read-access"
  }
}

resource "aws_s3_bucket_acl" "public_acl" {
  bucket = aws_s3_bucket.public_bucket.id
  acl    = "public-read" # PUBLIC ACCESS - HIGH SEVERITY
}

# Note: No aws_s3_bucket_public_access_block to prevent public access

# Expected Security Findings:
# 1. Security group with SSH from 0.0.0.0/0 - HIGH
# 2. Security group with all traffic from 0.0.0.0/0 - CRITICAL
# 3. Database security group exposed to internet - HIGH
# 4. IAM policy with wildcard permissions - MEDIUM/HIGH
# 5. S3 policy with wildcard on all buckets - MEDIUM
# 6. IAM role assumable by anyone - MEDIUM
# 7. S3 bucket with public read access - HIGH
#
# Expected Compliance Status: NON_COMPLIANT
# Expected Compliance Gaps:
# - Network security: Overly permissive security groups
# - Access control: Wildcard IAM permissions
# - Controls failing: CC6.1 (access control), CC6.6 (network security)
# - Evidence tasks failing: ET-071 (firewall configurations)
