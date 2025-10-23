# Invalid Terraform syntax for error testing
terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Missing closing brace - syntax error
resource "aws_s3_bucket" "invalid_bucket" {
  bucket = "test-invalid-bucket"
  
  tags = {
    Environment = "test"
    Purpose = "syntax-error-testing"
  # Missing closing brace here

# Invalid resource reference
resource "aws_s3_bucket_policy" "invalid_policy" {
  bucket = aws_s3_bucket.nonexistent_bucket.id  # This bucket doesn't exist
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = "*"  # Invalid - should be an object
        Action = "s3:GetObject"
        Resource = "${aws_s3_bucket.invalid_bucket.arn}/*"
      }
    ]
  })
}

# Invalid interpolation syntax
resource "aws_instance" "invalid_instance" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  
  # Invalid interpolation - using old syntax
  tags = {
    Name = "${var.nonexistent_variable}"  # Variable doesn't exist
  }
}

# Circular dependency
resource "aws_security_group" "sg1" {
  name   = "sg1"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.sg2.id]  # References sg2
  }
}

resource "aws_security_group" "sg2" {
  name   = "sg2"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.sg1.id]  # References sg1 - circular!
  }
}

# Invalid attribute reference
resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  subnet_id     = aws_subnet.main.invalid_attribute  # 'invalid_attribute' doesn't exist
}

# Missing required argument
resource "aws_db_instance" "example" {
  # Missing required arguments like identifier, engine, instance_class, etc.
  allocated_storage = 10
}

# Invalid HCL syntax - malformed JSON
resource "aws_iam_policy" "invalid_json" {
  name = "invalid-policy"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          # Missing comma here
          "s3:PutObject"
        ]
        Resource = "*"
        # Missing closing brace
    ]
  })
}

# Invalid provider configuration
provider "nonexistent_provider" {
  region = "us-west-2"
}

# Multiple syntax errors in one block
resource "aws_lambda_function" "broken_lambda" {
  function_name = "broken-function"  # Invalid attribute name - should be 'function_name'
  role          = aws_iam_role.nonexistent_role.arn  # Role doesn't exist
  handler       = "index.handler"
  runtime       = "python3.9"
  
  # Invalid nested block syntax
  environment {
    variables = {
      KEY1 = "value1"
      KEY2 = var.undefined_variable  # Variable not defined
      # Missing closing brace for variables
    # Missing closing brace for environment
  
  # Missing closing brace for resource
}