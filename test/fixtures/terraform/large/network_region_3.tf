# Network Infrastructure - Region 3
# SOC2 Controls: CC6.6, CC7.1

resource "aws_vpc" "region_3" {
  cidr_block           = "10.3.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "vpc-region-3"
    Environment = "prod"
    Region      = "us-east-3"
  }
}

resource "aws_subnet" "region_3_public_a" {
  vpc_id            = aws_vpc.region_3.id
  cidr_block        = "10.3.1.0/24"
  availability_zone = "us-east-3a"

  tags = {
    Name   = "public-3a"
    Tier   = "public"
    Region = "us-east-3"
  }
}

resource "aws_subnet" "region_3_public_b" {
  vpc_id            = aws_vpc.region_3.id
  cidr_block        = "10.3.2.0/24"
  availability_zone = "us-east-3b"

  tags = {
    Name   = "public-3b"
    Tier   = "public"
    Region = "us-east-3"
  }
}

resource "aws_subnet" "region_3_private_a" {
  vpc_id            = aws_vpc.region_3.id
  cidr_block        = "10.3.10.0/24"
  availability_zone = "us-east-3a"

  tags = {
    Name   = "private-3a"
    Tier   = "private"
    Region = "us-east-3"
  }
}

resource "aws_subnet" "region_3_private_b" {
  vpc_id            = aws_vpc.region_3.id
  cidr_block        = "10.3.11.0/24"
  availability_zone = "us-east-3b"

  tags = {
    Name   = "private-3b"
    Tier   = "private"
    Region = "us-east-3"
  }
}

resource "aws_internet_gateway" "region_3" {
  vpc_id = aws_vpc.region_3.id

  tags = {
    Name   = "igw-region-3"
    Region = "us-east-3"
  }
}

resource "aws_nat_gateway" "region_3_a" {
  allocation_id = aws_eip.region_3_nat_a.id
  subnet_id     = aws_subnet.region_3_public_a.id

  tags = {
    Name   = "nat-3a"
    Region = "us-east-3"
  }
}

resource "aws_eip" "region_3_nat_a" {
  domain = "vpc"

  tags = {
    Name   = "nat-eip-3a"
    Region = "us-east-3"
  }
}

resource "aws_security_group" "region_3_web" {
  name        = "web-sg-region-3"
  description = "Security group for web tier in region 3"
  vpc_id      = aws_vpc.region_3.id

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
    Name   = "web-sg-3"
    Region = "us-east-3"
  }
}
