# Monitoring Resources Set 1
# SOC2 Controls: CC7.2, CC7.3

resource "aws_cloudtrail" "trail_1" {
  name                          = "cloudtrail-1"
  s3_bucket_name                = aws_s3_bucket.logs.id
  s3_key_prefix                 = "cloudtrail/1"
  include_global_service_events = true
  is_multi_region_trail         = true
  enable_log_file_validation    = true
  kms_key_id                    = aws_kms_key.cloudtrail.arn

  event_selector {
    read_write_type           = "All"
    include_management_events = true

    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::*/"]
    }
  }

  tags = {
    Name = "cloudtrail-1"
  }
}

resource "aws_cloudwatch_log_group" "app_1" {
  name              = "/aws/application/app-1"
  retention_in_days = 90
  kms_key_id        = aws_kms_key.cloudwatch.arn

  tags = {
    Name = "app-logs-1"
  }
}

resource "aws_cloudwatch_metric_alarm" "cpu_1" {
  alarm_name          = "high-cpu-alarm-1"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "120"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "CPU utilization alarm 1"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "cpu-alarm-1"
  }
}

resource "aws_cloudwatch_metric_alarm" "disk_1" {
  alarm_name          = "high-disk-alarm-1"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "DiskSpaceUtilization"
  namespace           = "System/Linux"
  period              = "300"
  statistic           = "Average"
  threshold           = "85"
  alarm_description   = "Disk space alarm 1"

  tags = {
    Name = "disk-alarm-1"
  }
}
