# Basic Infrastructure Terraform Configuration
# Purpose: Test fixtures for terraform indexer
# Coverage: Network security (CC6.6, CC7.1), Multi-AZ (SO2)

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# VPC with DNS support
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "main-vpc"
    Environment = "prod"
    ManagedBy   = "terraform"
  }
}

# Public subnets across multiple AZs (Multi-AZ configuration)
resource "aws_subnet" "public_1a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "us-east-1a"

  tags = {
    Name        = "public-subnet-1a"
    Environment = "prod"
    Tier        = "public"
  }
}

resource "aws_subnet" "public_1b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "us-east-1b"

  tags = {
    Name        = "public-subnet-1b"
    Environment = "prod"
    Tier        = "public"
  }
}

# Private subnets across multiple AZs
resource "aws_subnet" "private_1a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.10.0/24"
  availability_zone = "us-east-1a"

  tags = {
    Name        = "private-subnet-1a"
    Environment = "prod"
    Tier        = "private"
  }
}

resource "aws_subnet" "private_1b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.11.0/24"
  availability_zone = "us-east-1b"

  tags = {
    Name        = "private-subnet-1b"
    Environment = "prod"
    Tier        = "private"
  }
}

# Internet Gateway for public access
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name        = "main-igw"
    Environment = "prod"
  }
}

# NAT Gateways for private subnet internet access (Multi-AZ)
resource "aws_eip" "nat_1a" {
  domain = "vpc"

  tags = {
    Name        = "nat-eip-1a"
    Environment = "prod"
  }
}

resource "aws_eip" "nat_1b" {
  domain = "vpc"

  tags = {
    Name        = "nat-eip-1b"
    Environment = "prod"
  }
}

resource "aws_nat_gateway" "nat_1a" {
  allocation_id = aws_eip.nat_1a.id
  subnet_id     = aws_subnet.public_1a.id

  tags = {
    Name        = "nat-gateway-1a"
    Environment = "prod"
  }
}

resource "aws_nat_gateway" "nat_1b" {
  allocation_id = aws_eip.nat_1b.id
  subnet_id     = aws_subnet.public_1b.id

  tags = {
    Name        = "nat-gateway-1b"
    Environment = "prod"
  }
}

# Security group for web servers (network security control)
resource "aws_security_group" "web" {
  name        = "web-server-sg"
  description = "Security group for web servers"
  vpc_id      = aws_vpc.main.id

  # HTTPS from anywhere (acceptable for public web servers)
  ingress {
    description = "HTTPS from internet"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTP from anywhere (should redirect to HTTPS)
  ingress {
    description = "HTTP from internet"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # All outbound traffic allowed
  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "web-server-sg"
    Environment = "prod"
  }
}

# Security group for database servers (more restrictive)
resource "aws_security_group" "database" {
  name        = "database-sg"
  description = "Security group for database servers"
  vpc_id      = aws_vpc.main.id

  # PostgreSQL only from web servers
  ingress {
    description     = "PostgreSQL from web servers"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.web.id]
  }

  # No outbound internet access (best practice for databases)
  egress {
    description = "Outbound to VPC only"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["10.0.0.0/16"]
  }

  tags = {
    Name        = "database-sg"
    Environment = "prod"
  }
}

# Application Load Balancer (Multi-AZ, encryption in transit)
resource "aws_lb" "web" {
  name               = "web-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.web.id]
  subnets            = [aws_subnet.public_1a.id, aws_subnet.public_1b.id]

  enable_deletion_protection = true
  enable_http2               = true
  enable_cross_zone_load_balancing = true

  tags = {
    Name        = "web-alb"
    Environment = "prod"
  }
}

# Auto Scaling Group (Multi-AZ, high availability)
resource "aws_launch_template" "web" {
  name_prefix   = "web-server-"
  image_id      = "ami-0c55b159cbfafe1f0" # Amazon Linux 2
  instance_type = "t3.micro"

  vpc_security_group_ids = [aws_security_group.web.id]

  monitoring {
    enabled = true
  }

  metadata_options {
    http_tokens = "required" # IMDSv2 required (security best practice)
  }

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name        = "web-server"
      Environment = "prod"
    }
  }
}

resource "aws_autoscaling_group" "web" {
  name                = "web-asg"
  vpc_zone_identifier = [aws_subnet.private_1a.id, aws_subnet.private_1b.id]
  target_group_arns   = [aws_lb_target_group.web.arn]
  health_check_type   = "ELB"
  health_check_grace_period = 300

  min_size         = 2
  max_size         = 10
  desired_capacity = 2

  launch_template {
    id      = aws_launch_template.web.id
    version = "$Latest"
  }

  tag {
    key                 = "Name"
    value               = "web-server"
    propagate_at_launch = true
  }

  tag {
    key                 = "Environment"
    value               = "prod"
    propagate_at_launch = true
  }
}

# Target group for load balancer
resource "aws_lb_target_group" "web" {
  name     = "web-tg"
  port     = 80
  protocol = "HTTP"
  vpc_id   = aws_vpc.main.id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 5
    interval            = 30
    path                = "/health"
    matcher             = "200"
  }

  tags = {
    Name        = "web-target-group"
    Environment = "prod"
  }
}
