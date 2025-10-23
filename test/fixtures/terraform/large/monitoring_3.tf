# Monitoring Resources Set 3
# SOC2 Controls: CC7.2, CC7.3

resource "aws_cloudtrail" "trail_3" {
  name                          = "cloudtrail-3"
  s3_bucket_name                = aws_s3_bucket.logs.id
  s3_key_prefix                 = "cloudtrail/3"
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
    Name = "cloudtrail-3"
  }
}

resource "aws_cloudwatch_log_group" "app_3" {
  name              = "/aws/application/app-3"
  retention_in_days = 90
  kms_key_id        = aws_kms_key.cloudwatch.arn

  tags = {
    Name = "app-logs-3"
  }
}

resource "aws_cloudwatch_metric_alarm" "cpu_3" {
  alarm_name          = "high-cpu-alarm-3"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "120"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "CPU utilization alarm 3"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "cpu-alarm-3"
  }
}

resource "aws_cloudwatch_metric_alarm" "disk_3" {
  alarm_name          = "high-disk-alarm-3"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "DiskSpaceUtilization"
  namespace           = "System/Linux"
  period              = "300"
  statistic           = "Average"
  threshold           = "85"
  alarm_description   = "Disk space alarm 3"

  tags = {
    Name = "disk-alarm-3"
  }
}
