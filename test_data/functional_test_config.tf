# Functional Test Configuration
# Simple terraform configuration for CLI functional testing

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Simple S3 bucket for testing
resource "aws_s3_bucket" "functional_test" {
  bucket = "grctool-functional-test-${random_id.bucket_suffix.hex}"

  tags = {
    Purpose = "Functional Testing"
    Tool    = "grctool"
    Type    = "test-data"
  }
}

# Bucket encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "functional_test" {
  bucket = aws_s3_bucket.functional_test.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Block public access
resource "aws_s3_bucket_public_access_block" "functional_test" {
  bucket = aws_s3_bucket.functional_test.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Versioning
resource "aws_s3_bucket_versioning" "functional_test" {
  bucket = aws_s3_bucket.functional_test.id
  versioning_configuration {
    status = "Enabled"
  }
}

# Simple IAM role for testing
resource "aws_iam_role" "functional_test" {
  name = "grctool-functional-test-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Purpose = "Functional Testing"
    Tool    = "grctool"
  }
}

# Policy attachment
resource "aws_iam_role_policy_attachment" "functional_test" {
  role       = aws_iam_role.functional_test.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Random suffix for unique naming
resource "random_id" "bucket_suffix" {
  byte_length = 4
}

# Output for testing
output "bucket_name" {
  description = "Name of the functional test bucket"
  value       = aws_s3_bucket.functional_test.bucket
}

output "role_arn" {
  description = "ARN of the functional test role"
  value       = aws_iam_role.functional_test.arn
}
