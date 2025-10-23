# Network Infrastructure - Region 1
# SOC2 Controls: CC6.6, CC7.1

resource "aws_vpc" "region_1" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "vpc-region-1"
    Environment = "prod"
    Region      = "us-east-1"
  }
}

resource "aws_subnet" "region_1_public_a" {
  vpc_id            = aws_vpc.region_1.id
  cidr_block        = "10.1.1.0/24"
  availability_zone = "us-east-1a"

  tags = {
    Name   = "public-1a"
    Tier   = "public"
    Region = "us-east-1"
  }
}

resource "aws_subnet" "region_1_public_b" {
  vpc_id            = aws_vpc.region_1.id
  cidr_block        = "10.1.2.0/24"
  availability_zone = "us-east-1b"

  tags = {
    Name   = "public-1b"
    Tier   = "public"
    Region = "us-east-1"
  }
}

resource "aws_subnet" "region_1_private_a" {
  vpc_id            = aws_vpc.region_1.id
  cidr_block        = "10.1.10.0/24"
  availability_zone = "us-east-1a"

  tags = {
    Name   = "private-1a"
    Tier   = "private"
    Region = "us-east-1"
  }
}

resource "aws_subnet" "region_1_private_b" {
  vpc_id            = aws_vpc.region_1.id
  cidr_block        = "10.1.11.0/24"
  availability_zone = "us-east-1b"

  tags = {
    Name   = "private-1b"
    Tier   = "private"
    Region = "us-east-1"
  }
}

resource "aws_internet_gateway" "region_1" {
  vpc_id = aws_vpc.region_1.id

  tags = {
    Name   = "igw-region-1"
    Region = "us-east-1"
  }
}

resource "aws_nat_gateway" "region_1_a" {
  allocation_id = aws_eip.region_1_nat_a.id
  subnet_id     = aws_subnet.region_1_public_a.id

  tags = {
    Name   = "nat-1a"
    Region = "us-east-1"
  }
}

resource "aws_eip" "region_1_nat_a" {
  domain = "vpc"

  tags = {
    Name   = "nat-eip-1a"
    Region = "us-east-1"
  }
}

resource "aws_security_group" "region_1_web" {
  name        = "web-sg-region-1"
  description = "Security group for web tier in region 1"
  vpc_id      = aws_vpc.region_1.id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name   = "web-sg-1"
    Region = "us-east-1"
  }
}
