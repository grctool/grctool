# Network Infrastructure - Region 2
# SOC2 Controls: CC6.6, CC7.1

resource "aws_vpc" "region_2" {
  cidr_block           = "10.2.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "vpc-region-2"
    Environment = "prod"
    Region      = "us-east-2"
  }
}

resource "aws_subnet" "region_2_public_a" {
  vpc_id            = aws_vpc.region_2.id
  cidr_block        = "10.2.1.0/24"
  availability_zone = "us-east-2a"

  tags = {
    Name   = "public-2a"
    Tier   = "public"
    Region = "us-east-2"
  }
}

resource "aws_subnet" "region_2_public_b" {
  vpc_id            = aws_vpc.region_2.id
  cidr_block        = "10.2.2.0/24"
  availability_zone = "us-east-2b"

  tags = {
    Name   = "public-2b"
    Tier   = "public"
    Region = "us-east-2"
  }
}

resource "aws_subnet" "region_2_private_a" {
  vpc_id            = aws_vpc.region_2.id
  cidr_block        = "10.2.10.0/24"
  availability_zone = "us-east-2a"

  tags = {
    Name   = "private-2a"
    Tier   = "private"
    Region = "us-east-2"
  }
}

resource "aws_subnet" "region_2_private_b" {
  vpc_id            = aws_vpc.region_2.id
  cidr_block        = "10.2.11.0/24"
  availability_zone = "us-east-2b"

  tags = {
    Name   = "private-2b"
    Tier   = "private"
    Region = "us-east-2"
  }
}

resource "aws_internet_gateway" "region_2" {
  vpc_id = aws_vpc.region_2.id

  tags = {
    Name   = "igw-region-2"
    Region = "us-east-2"
  }
}

resource "aws_nat_gateway" "region_2_a" {
  allocation_id = aws_eip.region_2_nat_a.id
  subnet_id     = aws_subnet.region_2_public_a.id

  tags = {
    Name   = "nat-2a"
    Region = "us-east-2"
  }
}

resource "aws_eip" "region_2_nat_a" {
  domain = "vpc"

  tags = {
    Name   = "nat-eip-2a"
    Region = "us-east-2"
  }
}

resource "aws_security_group" "region_2_web" {
  name        = "web-sg-region-2"
  description = "Security group for web tier in region 2"
  vpc_id      = aws_vpc.region_2.id

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
    Name   = "web-sg-2"
    Region = "us-east-2"
  }
}
