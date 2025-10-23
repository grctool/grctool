# Network security configuration for compliance
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# VPC with proper CIDR planning
resource "aws_vpc" "security_vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "security-vpc"
    Environment = "production"
    Compliance  = "SOC2"
  }
}

# Internet Gateway
resource "aws_internet_gateway" "security_igw" {
  vpc_id = aws_vpc.security_vpc.id

  tags = {
    Name = "security-internet-gateway"
  }
}

# Public subnets for load balancers (multi-AZ)
resource "aws_subnet" "public_1" {
  vpc_id                  = aws_vpc.security_vpc.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = data.aws_availability_zones.available.names[0]
  map_public_ip_on_launch = true

  tags = {
    Name = "security-public-subnet-1"
    Type = "Public"
  }
}

resource "aws_subnet" "public_2" {
  vpc_id                  = aws_vpc.security_vpc.id
  cidr_block              = "10.0.2.0/24"
  availability_zone       = data.aws_availability_zones.available.names[1]
  map_public_ip_on_launch = true

  tags = {
    Name = "security-public-subnet-2"
    Type = "Public"
  }
}

# Private subnets for application servers (multi-AZ)
resource "aws_subnet" "private_1" {
  vpc_id            = aws_vpc.security_vpc.id
  cidr_block        = "10.0.10.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "security-private-subnet-1"
    Type = "Private"
  }
}

resource "aws_subnet" "private_2" {
  vpc_id            = aws_vpc.security_vpc.id
  cidr_block        = "10.0.11.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "security-private-subnet-2"
    Type = "Private"
  }
}

resource "aws_subnet" "private_3" {
  vpc_id            = aws_vpc.security_vpc.id
  cidr_block        = "10.0.12.0/24"
  availability_zone = data.aws_availability_zones.available.names[2]

  tags = {
    Name = "security-private-subnet-3"
    Type = "Private"
  }
}

# Database subnets (isolated)
resource "aws_subnet" "database_1" {
  vpc_id            = aws_vpc.security_vpc.id
  cidr_block        = "10.0.20.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "security-database-subnet-1"
    Type = "Database"
  }
}

resource "aws_subnet" "database_2" {
  vpc_id            = aws_vpc.security_vpc.id
  cidr_block        = "10.0.21.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "security-database-subnet-2"
    Type = "Database"
  }
}

# NAT Gateways for private subnet internet access
resource "aws_eip" "nat_1" {
  domain = "vpc"
  depends_on = [aws_internet_gateway.security_igw]

  tags = {
    Name = "security-nat-eip-1"
  }
}

resource "aws_eip" "nat_2" {
  domain = "vpc"
  depends_on = [aws_internet_gateway.security_igw]

  tags = {
    Name = "security-nat-eip-2"
  }
}

resource "aws_nat_gateway" "nat_1" {
  allocation_id = aws_eip.nat_1.id
  subnet_id     = aws_subnet.public_1.id

  tags = {
    Name = "security-nat-gateway-1"
  }

  depends_on = [aws_internet_gateway.security_igw]
}

resource "aws_nat_gateway" "nat_2" {
  allocation_id = aws_eip.nat_2.id
  subnet_id     = aws_subnet.public_2.id

  tags = {
    Name = "security-nat-gateway-2"
  }

  depends_on = [aws_internet_gateway.security_igw]
}

# Route tables
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.security_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.security_igw.id
  }

  tags = {
    Name = "security-public-route-table"
  }
}

resource "aws_route_table" "private_1" {
  vpc_id = aws_vpc.security_vpc.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.nat_1.id
  }

  tags = {
    Name = "security-private-route-table-1"
  }
}

resource "aws_route_table" "private_2" {
  vpc_id = aws_vpc.security_vpc.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.nat_2.id
  }

  tags = {
    Name = "security-private-route-table-2"
  }
}

# Database route table (no internet access)
resource "aws_route_table" "database" {
  vpc_id = aws_vpc.security_vpc.id

  tags = {
    Name = "security-database-route-table"
  }
}

# Route table associations
resource "aws_route_table_association" "public_1" {
  subnet_id      = aws_subnet.public_1.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "public_2" {
  subnet_id      = aws_subnet.public_2.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "private_1" {
  subnet_id      = aws_subnet.private_1.id
  route_table_id = aws_route_table.private_1.id
}

resource "aws_route_table_association" "private_2" {
  subnet_id      = aws_subnet.private_2.id
  route_table_id = aws_route_table.private_2.id
}

resource "aws_route_table_association" "private_3" {
  subnet_id      = aws_subnet.private_3.id
  route_table_id = aws_route_table.private_1.id # Shares NAT with private_1
}

resource "aws_route_table_association" "database_1" {
  subnet_id      = aws_subnet.database_1.id
  route_table_id = aws_route_table.database.id
}

resource "aws_route_table_association" "database_2" {
  subnet_id      = aws_subnet.database_2.id
  route_table_id = aws_route_table.database.id
}

# Security Groups with least privilege
resource "aws_security_group" "alb_sg" {
  name        = "security-alb-sg"
  description = "Security group for Application Load Balancer"
  vpc_id      = aws_vpc.security_vpc.id

  ingress {
    description = "HTTPS from internet"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTP from internet (redirect to HTTPS)"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description     = "To application servers"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.app_sg.id]
  }

  tags = {
    Name    = "security-alb-sg"
    Purpose = "load-balancer"
  }
}

resource "aws_security_group" "app_sg" {
  name        = "security-app-sg"
  description = "Security group for application servers"
  vpc_id      = aws_vpc.security_vpc.id

  ingress {
    description     = "From load balancer"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb_sg.id]
  }

  ingress {
    description     = "SSH from bastion"
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.bastion_sg.id]
  }

  egress {
    description     = "To database"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.db_sg.id]
  }

  egress {
    description = "HTTPS to internet"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name    = "security-app-sg"
    Purpose = "application-servers"
  }
}

resource "aws_security_group" "db_sg" {
  name        = "security-db-sg"
  description = "Security group for database servers"
  vpc_id      = aws_vpc.security_vpc.id

  ingress {
    description     = "PostgreSQL from app servers"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app_sg.id]
  }

  # No egress rules - database should not initiate outbound connections

  tags = {
    Name    = "security-db-sg"
    Purpose = "database-servers"
  }
}

resource "aws_security_group" "bastion_sg" {
  name        = "security-bastion-sg"
  description = "Security group for bastion host"
  vpc_id      = aws_vpc.security_vpc.id

  ingress {
    description = "SSH from specific IP ranges"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.allowed_ssh_cidrs
  }

  egress {
    description = "SSH to private instances"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.security_vpc.cidr_block]
  }

  tags = {
    Name    = "security-bastion-sg"
    Purpose = "bastion-host"
  }
}

# Network ACLs for additional security
resource "aws_network_acl" "database_nacl" {
  vpc_id     = aws_vpc.security_vpc.id
  subnet_ids = [aws_subnet.database_1.id, aws_subnet.database_2.id]

  # Allow inbound PostgreSQL from private subnets
  ingress {
    protocol   = "tcp"
    rule_no    = 100
    action     = "allow"
    cidr_block = "10.0.10.0/24"
    from_port  = 5432
    to_port    = 5432
  }

  ingress {
    protocol   = "tcp"
    rule_no    = 101
    action     = "allow"
    cidr_block = "10.0.11.0/24"
    from_port  = 5432
    to_port    = 5432
  }

  # Allow return traffic
  ingress {
    protocol   = "tcp"
    rule_no    = 200
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 1024
    to_port    = 65535
  }

  # Allow outbound responses
  egress {
    protocol   = "tcp"
    rule_no    = 100
    action     = "allow"
    cidr_block = "10.0.10.0/24"
    from_port  = 1024
    to_port    = 65535
  }

  egress {
    protocol   = "tcp"
    rule_no    = 101
    action     = "allow"
    cidr_block = "10.0.11.0/24"
    from_port  = 1024
    to_port    = 65535
  }

  tags = {
    Name = "security-database-nacl"
  }
}

# VPC Flow Logs for monitoring
resource "aws_flow_log" "vpc_flow_log" {
  iam_role_arn    = aws_iam_role.flow_log_role.arn
  log_destination = aws_cloudwatch_log_group.vpc_flow_log.arn
  traffic_type    = "ALL"
  vpc_id          = aws_vpc.security_vpc.id

  tags = {
    Name = "security-vpc-flow-log"
  }
}

resource "aws_cloudwatch_log_group" "vpc_flow_log" {
  name              = "/aws/vpc/flowlogs"
  retention_in_days = 90

  tags = {
    Environment = "production"
    Purpose     = "vpc-flow-logs"
  }
}

resource "aws_iam_role" "flow_log_role" {
  name = "flow-log-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "vpc-flow-logs.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "flow_log_policy" {
  name = "flow-log-policy"
  role = aws_iam_role.flow_log_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Effect   = "Allow"
        Resource = "*"
      }
    ]
  })
}

# Variables
variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed for SSH access to bastion"
  type        = list(string)
  default     = ["10.0.0.0/8"] # Default to private networks only
}

# Data sources
data "aws_availability_zones" "available" {
  state = "available"
}