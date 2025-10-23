# Monitoring Resources Set 4
# SOC2 Controls: CC7.2, CC7.3

resource "aws_cloudtrail" "trail_4" {
  name                          = "cloudtrail-4"
  s3_bucket_name                = aws_s3_bucket.logs.id
  s3_key_prefix                 = "cloudtrail/4"
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
    Name = "cloudtrail-4"
  }
}

resource "aws_cloudwatch_log_group" "app_4" {
  name              = "/aws/application/app-4"
  retention_in_days = 90
  kms_key_id        = aws_kms_key.cloudwatch.arn

  tags = {
    Name = "app-logs-4"
  }
}

resource "aws_cloudwatch_metric_alarm" "cpu_4" {
  alarm_name          = "high-cpu-alarm-4"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "120"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "CPU utilization alarm 4"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "cpu-alarm-4"
  }
}

resource "aws_cloudwatch_metric_alarm" "disk_4" {
  alarm_name          = "high-disk-alarm-4"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "DiskSpaceUtilization"
  namespace           = "System/Linux"
  period              = "300"
  statistic           = "Average"
  threshold           = "85"
  alarm_description   = "Disk space alarm 4"

  tags = {
    Name = "disk-alarm-4"
  }
}
