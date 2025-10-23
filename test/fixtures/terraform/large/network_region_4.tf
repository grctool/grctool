# Network Infrastructure - Region 4
# SOC2 Controls: CC6.6, CC7.1

resource "aws_vpc" "region_4" {
  cidr_block           = "10.4.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "vpc-region-4"
    Environment = "prod"
    Region      = "us-east-4"
  }
}

resource "aws_subnet" "region_4_public_a" {
  vpc_id            = aws_vpc.region_4.id
  cidr_block        = "10.4.1.0/24"
  availability_zone = "us-east-4a"

  tags = {
    Name   = "public-4a"
    Tier   = "public"
    Region = "us-east-4"
  }
}

resource "aws_subnet" "region_4_public_b" {
  vpc_id            = aws_vpc.region_4.id
  cidr_block        = "10.4.2.0/24"
  availability_zone = "us-east-4b"

  tags = {
    Name   = "public-4b"
    Tier   = "public"
    Region = "us-east-4"
  }
}

resource "aws_subnet" "region_4_private_a" {
  vpc_id            = aws_vpc.region_4.id
  cidr_block        = "10.4.10.0/24"
  availability_zone = "us-east-4a"

  tags = {
    Name   = "private-4a"
    Tier   = "private"
    Region = "us-east-4"
  }
}

resource "aws_subnet" "region_4_private_b" {
  vpc_id            = aws_vpc.region_4.id
  cidr_block        = "10.4.11.0/24"
  availability_zone = "us-east-4b"

  tags = {
    Name   = "private-4b"
    Tier   = "private"
    Region = "us-east-4"
  }
}

resource "aws_internet_gateway" "region_4" {
  vpc_id = aws_vpc.region_4.id

  tags = {
    Name   = "igw-region-4"
    Region = "us-east-4"
  }
}

resource "aws_nat_gateway" "region_4_a" {
  allocation_id = aws_eip.region_4_nat_a.id
  subnet_id     = aws_subnet.region_4_public_a.id

  tags = {
    Name   = "nat-4a"
    Region = "us-east-4"
  }
}

resource "aws_eip" "region_4_nat_a" {
  domain = "vpc"

  tags = {
    Name   = "nat-eip-4a"
    Region = "us-east-4"
  }
}

resource "aws_security_group" "region_4_web" {
  name        = "web-sg-region-4"
  description = "Security group for web tier in region 4"
  vpc_id      = aws_vpc.region_4.id

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
    Name   = "web-sg-4"
    Region = "us-east-4"
  }
}
