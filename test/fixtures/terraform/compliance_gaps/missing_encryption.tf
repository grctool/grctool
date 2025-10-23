# Compliance Gaps: Missing Encryption
# Purpose: Test fixtures for detecting encryption compliance gaps
# Expected Findings: HIGH severity - missing encryption at rest and in transit

# S3 bucket WITHOUT encryption (COMPLIANCE GAP - HIGH)
resource "aws_s3_bucket" "unencrypted_data" {
  bucket = "unencrypted-data-bucket"

  tags = {
    Name        = "unencrypted-data"
    Environment = "prod"
    Issue       = "no-encryption-at-rest"
  }
}

# Note: No aws_s3_bucket_server_side_encryption_configuration defined
# This should trigger a HIGH severity finding: "S3 bucket does not have server-side encryption configured"

# RDS instance WITHOUT encryption (COMPLIANCE GAP - HIGH)
resource "aws_rds_instance" "unencrypted_db" {
  identifier           = "unencrypted-database"
  engine               = "postgres"
  engine_version       = "14.6"
  instance_class       = "db.t3.micro"
  allocated_storage    = 20
  username             = "dbadmin"
  password             = "insecure-password"
  skip_final_snapshot  = true

  # storage_encrypted = false (default)
  # This should trigger a HIGH severity finding: "RDS instance does not have encryption at rest enabled"

  tags = {
    Name        = "unencrypted-db"
    Environment = "prod"
    Issue       = "no-encryption-at-rest"
  }
}

# Load balancer with HTTP only (COMPLIANCE GAP - HIGH)
resource "aws_lb" "http_only" {
  name               = "http-only-lb"
  internal           = false
  load_balancer_type = "application"
  subnets            = ["subnet-abc123", "subnet-def456"]

  tags = {
    Name        = "http-only-lb"
    Environment = "prod"
    Issue       = "no-encryption-in-transit"
  }
}

# HTTP listener without HTTPS redirect (COMPLIANCE GAP - HIGH)
resource "aws_lb_listener" "http_no_redirect" {
  load_balancer_arn = aws_lb.http_only.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "forward"
    target_group_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/web/abc123"
  }
}

# Note: No HTTPS listener, no redirect from HTTP to HTTPS
# This should trigger: "All API and web traffic uses TLS 1.2 or higher" - FAILED

# EBS volume without encryption (COMPLIANCE GAP - HIGH)
resource "aws_ebs_volume" "unencrypted_volume" {
  availability_zone = "us-east-1a"
  size              = 40

  # encrypted = false (default)
  # This should trigger a HIGH severity finding

  tags = {
    Name        = "unencrypted-volume"
    Environment = "prod"
    Issue       = "no-ebs-encryption"
  }
}

# Expected Security Findings:
# 1. S3 bucket without encryption - HIGH
# 2. RDS instance without encryption - HIGH
# 3. Load balancer without HTTPS - HIGH
# 4. EBS volume without encryption - HIGH
#
# Expected Compliance Status: NON_COMPLIANT
# Expected Compliance Gaps:
# - Encryption domain: Missing encryption configurations
# - Controls failing: CC6.8 (Data Protection)
# - Evidence tasks failing: ET-021 (encryption in transit), ET-023 (encryption at rest)
