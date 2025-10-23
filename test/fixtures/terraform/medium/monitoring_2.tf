# Monitoring Resources - File ${i}
# SOC2 Controls: CC7.2, CC7.3

resource "aws_cloudtrail" "main_${i}" {
  name                          = "main-trail-${i}"
  s3_bucket_name                = aws_s3_bucket.logs.id
  include_global_service_events = true
  is_multi_region_trail         = true
  enable_log_file_validation    = true

  event_selector {
    read_write_type           = "All"
    include_management_events = true
  }

  tags = {
    Name = "main-trail-${i}"
  }
}

resource "aws_cloudwatch_log_group" "app_${i}" {
  name              = "/aws/app-${i}"
  retention_in_days = 90

  tags = {
    Name = "app-logs-${i}"
  }
}

resource "aws_cloudwatch_metric_alarm" "high_cpu_${i}" {
  alarm_name          = "high-cpu-${i}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "120"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors ec2 cpu utilization"

  tags = {
    Name = "high-cpu-alarm-${i}"
  }
}
