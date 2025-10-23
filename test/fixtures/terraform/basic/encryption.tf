# Encryption and Data Protection Configuration
# Purpose: Test fixtures for encryption evidence (ET-021, ET-023)
# Coverage: Encryption at rest/transit (CC6.8), Key management

# KMS key for data encryption at rest
resource "aws_kms_key" "main" {
  description             = "Main KMS key for data encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name        = "main-kms-key"
    Environment = "prod"
    Purpose     = "data-encryption"
  }
}

resource "aws_kms_alias" "main" {
  name          = "alias/main-key"
  target_key_id = aws_kms_key.main.key_id
}

# S3 bucket with encryption at rest
resource "aws_s3_bucket" "data" {
  bucket = "example-data-bucket-prod"

  tags = {
    Name        = "data-bucket"
    Environment = "prod"
    Sensitivity = "high"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "data" {
  bucket = aws_s3_bucket.data.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.main.arn
    }
    bucket_key_enabled = true
  }
}

# S3 bucket versioning (backup/disaster recovery)
resource "aws_s3_bucket_versioning" "data" {
  bucket = aws_s3_bucket.data.id

  versioning_configuration {
    status = "Enabled"
  }
}

# Block public access (security best practice)
resource "aws_s3_bucket_public_access_block" "data" {
  bucket = aws_s3_bucket.data.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# S3 bucket for logs
resource "aws_s3_bucket" "logs" {
  bucket = "example-logs-bucket-prod"

  tags = {
    Name        = "logs-bucket"
    Environment = "prod"
    Purpose     = "audit-logs"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "logs" {
  bucket = aws_s3_bucket.logs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# RDS cluster with encryption at rest (Multi-AZ)
resource "aws_rds_cluster" "main" {
  cluster_identifier      = "main-db-cluster"
  engine                  = "aurora-postgresql"
  engine_version          = "14.6"
  database_name           = "appdb"
  master_username         = "dbadmin"
  master_password         = "change-me-in-production" # Should use secrets manager
  backup_retention_period = 7
  preferred_backup_window = "03:00-04:00"

  # Encryption at rest
  storage_encrypted = true
  kms_key_id        = aws_kms_key.main.arn

  # Multi-AZ for high availability
  availability_zones = ["us-east-1a", "us-east-1b"]

  # Security
  vpc_security_group_ids = [aws_security_group.database.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name

  # Deletion protection
  deletion_protection = true
  skip_final_snapshot = false
  final_snapshot_identifier = "main-db-cluster-final-snapshot"

  # Enhanced monitoring
  enabled_cloudwatch_logs_exports = ["postgresql"]

  tags = {
    Name        = "main-db-cluster"
    Environment = "prod"
  }
}

resource "aws_db_subnet_group" "main" {
  name       = "main-db-subnet-group"
  subnet_ids = [aws_subnet.private_1a.id, aws_subnet.private_1b.id]

  tags = {
    Name        = "main-db-subnet-group"
    Environment = "prod"
  }
}

# RDS cluster instances (Multi-AZ)
resource "aws_rds_cluster_instance" "main_1a" {
  identifier         = "main-db-instance-1a"
  cluster_identifier = aws_rds_cluster.main.id
  instance_class     = "db.t3.medium"
  engine             = aws_rds_cluster.main.engine
  engine_version     = aws_rds_cluster.main.engine_version

  # Performance insights for monitoring
  performance_insights_enabled = true
  performance_insights_kms_key_id = aws_kms_key.main.arn

  # Monitoring
  monitoring_interval = 60
  monitoring_role_arn = aws_iam_role.rds_monitoring.arn

  tags = {
    Name        = "main-db-instance-1a"
    Environment = "prod"
    AZ          = "us-east-1a"
  }
}

resource "aws_rds_cluster_instance" "main_1b" {
  identifier         = "main-db-instance-1b"
  cluster_identifier = aws_rds_cluster.main.id
  instance_class     = "db.t3.medium"
  engine             = aws_rds_cluster.main.engine
  engine_version     = aws_rds_cluster.main.engine_version

  # Performance insights for monitoring
  performance_insights_enabled = true
  performance_insights_kms_key_id = aws_kms_key.main.arn

  # Monitoring
  monitoring_interval = 60
  monitoring_role_arn = aws_iam_role.rds_monitoring.arn

  tags = {
    Name        = "main-db-instance-1b"
    Environment = "prod"
    AZ          = "us-east-1b"
  }
}

# ACM certificate for TLS/SSL (encryption in transit)
resource "aws_acm_certificate" "web" {
  domain_name       = "example.com"
  validation_method = "DNS"

  subject_alternative_names = [
    "www.example.com",
    "*.example.com"
  ]

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name        = "web-certificate"
    Environment = "prod"
  }
}

# HTTPS listener for load balancer (encryption in transit)
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.web.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS-1-2-2017-01"
  certificate_arn   = aws_acm_certificate.web.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.web.arn
  }
}

# HTTP listener redirects to HTTPS
resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.web.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

# EBS encryption by default (security best practice)
resource "aws_ebs_encryption_by_default" "enabled" {
  enabled = true
}
