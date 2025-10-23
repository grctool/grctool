# Network Infrastructure - Region 5
# SOC2 Controls: CC6.6, CC7.1

resource "aws_vpc" "region_5" {
  cidr_block           = "10.5.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "vpc-region-5"
    Environment = "prod"
    Region      = "us-east-5"
  }
}

resource "aws_subnet" "region_5_public_a" {
  vpc_id            = aws_vpc.region_5.id
  cidr_block        = "10.5.1.0/24"
  availability_zone = "us-east-5a"

  tags = {
    Name   = "public-5a"
    Tier   = "public"
    Region = "us-east-5"
  }
}

resource "aws_subnet" "region_5_public_b" {
  vpc_id            = aws_vpc.region_5.id
  cidr_block        = "10.5.2.0/24"
  availability_zone = "us-east-5b"

  tags = {
    Name   = "public-5b"
    Tier   = "public"
    Region = "us-east-5"
  }
}

resource "aws_subnet" "region_5_private_a" {
  vpc_id            = aws_vpc.region_5.id
  cidr_block        = "10.5.10.0/24"
  availability_zone = "us-east-5a"

  tags = {
    Name   = "private-5a"
    Tier   = "private"
    Region = "us-east-5"
  }
}

resource "aws_subnet" "region_5_private_b" {
  vpc_id            = aws_vpc.region_5.id
  cidr_block        = "10.5.11.0/24"
  availability_zone = "us-east-5b"

  tags = {
    Name   = "private-5b"
    Tier   = "private"
    Region = "us-east-5"
  }
}

resource "aws_internet_gateway" "region_5" {
  vpc_id = aws_vpc.region_5.id

  tags = {
    Name   = "igw-region-5"
    Region = "us-east-5"
  }
}

resource "aws_nat_gateway" "region_5_a" {
  allocation_id = aws_eip.region_5_nat_a.id
  subnet_id     = aws_subnet.region_5_public_a.id

  tags = {
    Name   = "nat-5a"
    Region = "us-east-5"
  }
}

resource "aws_eip" "region_5_nat_a" {
  domain = "vpc"

  tags = {
    Name   = "nat-eip-5a"
    Region = "us-east-5"
  }
}

resource "aws_security_group" "region_5_web" {
  name        = "web-sg-region-5"
  description = "Security group for web tier in region 5"
  vpc_id      = aws_vpc.region_5.id

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
    Name   = "web-sg-5"
    Region = "us-east-5"
  }
}
