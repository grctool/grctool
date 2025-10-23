# IAM and Access Control Configuration
# Purpose: Test fixtures for access control evidence (ET-047)
# Coverage: Access control (CC6.1, CC6.3), Identity management, Least privilege

# IAM role for EC2 instances (application role)
resource "aws_iam_role" "app" {
  name        = "application-role"
  description = "IAM role for application EC2 instances"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "application-role"
    Environment = "prod"
  }
}

# IAM policy for S3 read access (least privilege)
resource "aws_iam_policy" "s3_read" {
  name        = "s3-read-only-policy"
  description = "Allows read-only access to specific S3 buckets"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.data.arn,
          "${aws_s3_bucket.data.arn}/*"
        ]
      }
    ]
  })

  tags = {
    Name        = "s3-read-only-policy"
    Environment = "prod"
  }
}

# IAM policy for CloudWatch metrics and logs
resource "aws_iam_policy" "cloudwatch" {
  name        = "cloudwatch-metrics-policy"
  description = "Allows writing metrics and logs to CloudWatch"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData",
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ]
        Resource = "*"
      }
    ]
  })

  tags = {
    Name        = "cloudwatch-metrics-policy"
    Environment = "prod"
  }
}

# Attach policies to application role
resource "aws_iam_role_policy_attachment" "app_s3" {
  role       = aws_iam_role.app.name
  policy_arn = aws_iam_policy.s3_read.arn
}

resource "aws_iam_role_policy_attachment" "app_cloudwatch" {
  role       = aws_iam_role.app.name
  policy_arn = aws_iam_policy.cloudwatch.arn
}

# IAM instance profile for EC2
resource "aws_iam_instance_profile" "app" {
  name = "application-instance-profile"
  role = aws_iam_role.app.name

  tags = {
    Name        = "application-instance-profile"
    Environment = "prod"
  }
}

# IAM role for RDS enhanced monitoring
resource "aws_iam_role" "rds_monitoring" {
  name        = "rds-monitoring-role"
  description = "IAM role for RDS enhanced monitoring"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "monitoring.rds.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "rds-monitoring-role"
    Environment = "prod"
  }
}

# Attach AWS managed policy for RDS monitoring
resource "aws_iam_role_policy_attachment" "rds_monitoring" {
  role       = aws_iam_role.rds_monitoring.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

# IAM role for CloudTrail to write to S3
resource "aws_iam_role" "cloudtrail" {
  name        = "cloudtrail-role"
  description = "IAM role for CloudTrail logging"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "cloudtrail-role"
    Environment = "prod"
  }
}

# IAM policy for CloudTrail S3 access
resource "aws_iam_policy" "cloudtrail_s3" {
  name        = "cloudtrail-s3-policy"
  description = "Allows CloudTrail to write to S3 bucket"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetBucketAcl"
        ]
        Resource = [
          aws_s3_bucket.logs.arn,
          "${aws_s3_bucket.logs.arn}/*"
        ]
      }
    ]
  })

  tags = {
    Name        = "cloudtrail-s3-policy"
    Environment = "prod"
  }
}

resource "aws_iam_role_policy_attachment" "cloudtrail_s3" {
  role       = aws_iam_role.cloudtrail.name
  policy_arn = aws_iam_policy.cloudtrail_s3.arn
}

# IAM user group for administrators (example)
resource "aws_iam_group" "admins" {
  name = "administrators"
  path = "/users/"
}

# IAM policy for admin group (admin access with conditions)
resource "aws_iam_group_policy" "admins" {
  name  = "admin-policy"
  group = aws_iam_group.admins.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "*"
        Resource = "*"
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = "true"
          }
        }
      }
    ]
  })
}

# IAM user group for developers (limited access)
resource "aws_iam_group" "developers" {
  name = "developers"
  path = "/users/"
}

# IAM policy for developer group (read-only production, full dev access)
resource "aws_iam_group_policy" "developers" {
  name  = "developer-policy"
  group = aws_iam_group.developers.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:Describe*",
          "s3:ListBucket",
          "s3:GetObject",
          "rds:Describe*",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:ListMetrics",
          "logs:GetLogEvents",
          "logs:FilterLogEvents"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:RequestedRegion" = "us-east-1"
          }
        }
      }
    ]
  })
}

# IAM password policy (security best practice)
resource "aws_iam_account_password_policy" "strict" {
  minimum_password_length        = 14
  require_lowercase_characters   = true
  require_uppercase_characters   = true
  require_numbers                = true
  require_symbols                = true
  allow_users_to_change_password = true
  max_password_age               = 90
  password_reuse_prevention      = 24
}
