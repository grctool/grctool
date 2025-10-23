# Database Instance 1
# SOC2 Controls: CC6.1, CC6.8

resource "aws_db_instance" "postgres_1" {
  identifier     = "postgres-db-1"
  engine         = "postgres"
  engine_version = "15.3"
  instance_class = "db.r6g.xlarge"

  allocated_storage     = 500
  max_allocated_storage = 5000
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.rds_1.arn

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds_1.id]

  backup_retention_period = 35
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  performance_insights_enabled    = true
  performance_insights_kms_key_id = aws_kms_key.rds_1.arn

  deletion_protection = true
  skip_final_snapshot = false

  tags = {
    Name        = "postgres-db-1"
    Environment = "prod"
    DBInstance  = "1"
  }
}

resource "aws_security_group" "rds_1" {
  name        = "rds-sg-1"
  description = "Security group for RDS 1"
  vpc_id      = aws_vpc.region_1.id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    security_groups = [aws_security_group.region_1_web.id]
  }

  tags = {
    Name = "rds-sg-1"
  }
}

resource "aws_db_subnet_group" "main" {
  name       = "main-db-subnet-group"
  subnet_ids = [aws_subnet.region_1_private_a.id, aws_subnet.region_1_private_b.id]

  tags = {
    Name = "main-db-subnet-group"
  }
}
