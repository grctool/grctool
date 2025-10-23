# Monitoring and Audit Logging Configuration
# Purpose: Test fixtures for monitoring and logging evidence
# Coverage: Audit logging (CC7.2, CC7.4), Monitoring, Compliance tracking

# CloudTrail for audit logging
resource "aws_cloudtrail" "main" {
  name                          = "main-audit-trail"
  s3_bucket_name                = aws_s3_bucket.logs.id
  include_global_service_events = true
  is_multi_region_trail         = true
  enable_log_file_validation    = true

  # Event selectors for data events
  event_selector {
    read_write_type           = "All"
    include_management_events = true

    data_resource {
      type   = "AWS::S3::Object"
      values = ["${aws_s3_bucket.data.arn}/"]
    }
  }

  # Send logs to CloudWatch for real-time monitoring
  cloud_watch_logs_group_arn = "${aws_cloudwatch_log_group.cloudtrail.arn}:*"
  cloud_watch_logs_role_arn  = aws_iam_role.cloudtrail.arn

  tags = {
    Name        = "main-audit-trail"
    Environment = "prod"
  }
}

# CloudWatch log group for CloudTrail
resource "aws_cloudwatch_log_group" "cloudtrail" {
  name              = "/aws/cloudtrail/main"
  retention_in_days = 365 # 1 year retention for compliance

  kms_key_id = aws_kms_key.main.arn

  tags = {
    Name        = "cloudtrail-logs"
    Environment = "prod"
  }
}

# CloudWatch log group for application logs
resource "aws_cloudwatch_log_group" "application" {
  name              = "/aws/application/main"
  retention_in_days = 90

  kms_key_id = aws_kms_key.main.arn

  tags = {
    Name        = "application-logs"
    Environment = "prod"
  }
}

# CloudWatch log group for VPC flow logs
resource "aws_cloudwatch_log_group" "vpc_flow_logs" {
  name              = "/aws/vpc/flow-logs"
  retention_in_days = 30

  tags = {
    Name        = "vpc-flow-logs"
    Environment = "prod"
  }
}

# IAM role for VPC Flow Logs
resource "aws_iam_role" "vpc_flow_logs" {
  name = "vpc-flow-logs-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "vpc-flow-logs.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "vpc-flow-logs-role"
    Environment = "prod"
  }
}

# IAM policy for VPC Flow Logs
resource "aws_iam_role_policy" "vpc_flow_logs" {
  name = "vpc-flow-logs-policy"
  role = aws_iam_role.vpc_flow_logs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Resource = "*"
      }
    ]
  })
}

# VPC Flow Logs
resource "aws_flow_log" "main" {
  iam_role_arn    = aws_iam_role.vpc_flow_logs.arn
  log_destination = aws_cloudwatch_log_group.vpc_flow_logs.arn
  traffic_type    = "ALL"
  vpc_id          = aws_vpc.main.id

  tags = {
    Name        = "main-vpc-flow-logs"
    Environment = "prod"
  }
}

# GuardDuty for threat detection
resource "aws_guardduty_detector" "main" {
  enable = true

  finding_publishing_frequency = "FIFTEEN_MINUTES"

  datasources {
    s3_logs {
      enable = true
    }
    kubernetes {
      audit_logs {
        enable = false # Enable if using EKS
      }
    }
  }

  tags = {
    Name        = "main-guardduty"
    Environment = "prod"
  }
}

# AWS Config for compliance tracking
resource "aws_config_configuration_recorder" "main" {
  name     = "main-config-recorder"
  role_arn = aws_iam_role.config.arn

  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }
}

# IAM role for AWS Config
resource "aws_iam_role" "config" {
  name = "config-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "config.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "config-role"
    Environment = "prod"
  }
}

# Attach AWS managed policy for Config
resource "aws_iam_role_policy_attachment" "config" {
  role       = aws_iam_role.config.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/ConfigRole"
}

# AWS Config delivery channel
resource "aws_config_delivery_channel" "main" {
  name           = "main-config-delivery"
  s3_bucket_name = aws_s3_bucket.logs.id

  depends_on = [aws_config_configuration_recorder.main]
}

# AWS Config recorder status
resource "aws_config_configuration_recorder_status" "main" {
  name       = aws_config_configuration_recorder.main.name
  is_enabled = true

  depends_on = [aws_config_delivery_channel.main]
}

# CloudWatch alarm for failed login attempts
resource "aws_cloudwatch_log_metric_filter" "failed_logins" {
  name           = "failed-login-attempts"
  log_group_name = aws_cloudwatch_log_group.cloudtrail.name
  pattern        = "{ $.eventName = ConsoleLogin && $.errorMessage = \"Failed authentication\" }"

  metric_transformation {
    name      = "FailedLoginAttempts"
    namespace = "Security"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "failed_logins" {
  alarm_name          = "high-failed-login-attempts"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "FailedLoginAttempts"
  namespace           = "Security"
  period              = "300"
  statistic           = "Sum"
  threshold           = "3"
  alarm_description   = "Alert on high number of failed login attempts"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.security_alerts.arn]

  tags = {
    Name        = "failed-login-alarm"
    Environment = "prod"
  }
}

# SNS topic for security alerts
resource "aws_sns_topic" "security_alerts" {
  name              = "security-alerts"
  kms_master_key_id = aws_kms_key.main.id

  tags = {
    Name        = "security-alerts"
    Environment = "prod"
  }
}

# CloudWatch dashboard for monitoring
resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "infrastructure-monitoring"

  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric"
        properties = {
          metrics = [
            ["AWS/ApplicationELB", "TargetResponseTime", { stat = "Average" }],
            [".", "RequestCount", { stat = "Sum" }],
            ["AWS/RDS", "DatabaseConnections", { stat = "Average" }]
          ]
          period = 300
          stat   = "Average"
          region = "us-east-1"
          title  = "Application Performance"
        }
      }
    ]
  })
}
