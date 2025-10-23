// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stubs

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// FileInfo represents file information - replicated from tools package to avoid import cycle
type FileInfo interface {
	Name() string
	Size() int64
	IsDir() bool
	ModTime() interface{} // Using interface{} to avoid time package dependency
}

// StubFileReader provides stub implementation for file operations
type StubFileReader struct {
	Files     map[string]string
	FileInfos map[string]*StubFileInfo
	Errors    map[string]error
}

// StubFileInfo implements the FileInfo interface
type StubFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

// Name returns the file name
func (f *StubFileInfo) Name() string {
	return f.name
}

// Size returns the file size
func (f *StubFileInfo) Size() int64 {
	return f.size
}

// IsDir returns whether this is a directory
func (f *StubFileInfo) IsDir() bool {
	return f.isDir
}

// ModTime returns the modification time
func (f *StubFileInfo) ModTime() interface{} {
	return f.modTime
}

// StubReadCloser wraps a string reader with Close method
type StubReadCloser struct {
	*strings.Reader
}

// Close implements io.Closer
func (s *StubReadCloser) Close() error {
	return nil
}

// NewStubFileReader creates a new stub file reader
func NewStubFileReader() *StubFileReader {
	return &StubFileReader{
		Files:     make(map[string]string),
		FileInfos: make(map[string]*StubFileInfo),
		Errors:    make(map[string]error),
	}
}

// Open returns a reader for the specified file
func (s *StubFileReader) Open(path string) (io.ReadCloser, error) {
	if err, ok := s.Errors[path]; ok {
		return nil, err
	}
	if content, ok := s.Files[path]; ok {
		return &StubReadCloser{strings.NewReader(content)}, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// Walk simulates walking a directory tree
func (s *StubFileReader) Walk(root string, walkFn func(path string, info FileInfo, err error) error) error {
	if err, ok := s.Errors[root]; ok {
		return err
	}

	// Walk through all files that start with the root path
	for path := range s.Files {
		if strings.HasPrefix(path, root) {
			info := s.FileInfos[path]
			if info == nil {
				// Create default info if not provided
				info = &StubFileInfo{
					name:    path,
					size:    int64(len(s.Files[path])),
					isDir:   false,
					modTime: time.Now(),
				}
			}
			if err := walkFn(path, info, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

// Glob returns paths matching a pattern
func (s *StubFileReader) Glob(pattern string) ([]string, error) {
	if err, ok := s.Errors[pattern]; ok {
		return nil, err
	}

	var matches []string
	for path := range s.Files {
		// Simple pattern matching - just check if pattern is contained in path
		// For a real implementation, you'd use filepath.Match or similar
		if strings.Contains(path, strings.TrimSuffix(pattern, "*")) {
			matches = append(matches, path)
		}
	}

	return matches, nil
}

// Helper methods to set up test data

// WithFile adds a file to the stub
func (s *StubFileReader) WithFile(path, content string) *StubFileReader {
	s.Files[path] = content
	return s
}

// WithFileInfo adds file info to the stub
func (s *StubFileReader) WithFileInfo(path string, info *StubFileInfo) *StubFileReader {
	s.FileInfos[path] = info
	return s
}

// WithError adds an error for a specific path
func (s *StubFileReader) WithError(path string, err error) *StubFileReader {
	s.Errors[path] = err
	return s
}

// CreateTestTerraformFile creates a test Terraform file content
func CreateTestTerraformFile() string {
	return `# Test Terraform configuration
resource "aws_s3_bucket" "example" {
  bucket = "my-test-bucket"
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
}

resource "aws_iam_role" "test_role" {
  name = "test-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_security_group" "web" {
  name_prefix = "web-"
  
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
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
}

module "vpc" {
  source = "./modules/vpc"
  
  cidr_block = "10.0.0.0/16"
  environment = "production"
}

data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  common_tags = {
    Environment = "production"
    Project     = "test-project"
  }
}`
}

// CreateTestTerraformSecurityFile creates a security-focused Terraform file
func CreateTestTerraformSecurityFile() string {
	return `# Security-focused Terraform configuration
resource "aws_kms_key" "example" {
  description             = "Example KMS key"
  deletion_window_in_days = 7
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::account-id:root"
        }
        Action   = "kms:*"
        Resource = "*"
      }
    ]
  })
}

resource "aws_s3_bucket_encryption" "example" {
  bucket = aws_s3_bucket.example.id
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.example.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

resource "aws_cloudtrail" "example" {
  name           = "example-cloudtrail"
  s3_bucket_name = aws_s3_bucket.cloudtrail.bucket
  
  enable_logging = true
  
  event_selector {
    read_write_type           = "All"
    include_management_events = true
    
    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::*/*"]
    }
  }
}`
}

// CreateTestTerraformModuleFile creates a Terraform file with modules
func CreateTestTerraformModuleFile() string {
	return `# Module-focused Terraform configuration
module "networking" {
  source = "terraform-aws-modules/vpc/aws"
  
  name = "example-vpc"
  cidr = "10.0.0.0/16"
  
  azs             = ["us-west-2a", "us-west-2b", "us-west-2c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
  
  enable_nat_gateway = true
  enable_vpn_gateway = true
}

module "database" {
  source = "./modules/rds"
  
  identifier = "example-db"
  
  engine         = "mysql"
  engine_version = "8.0"
  instance_class = "db.t3.micro"
  
  allocated_storage = 20
  storage_encrypted = true
  kms_key_id       = aws_kms_key.example.arn
  
  vpc_security_group_ids = [aws_security_group.database.id]
  subnet_group_name      = aws_db_subnet_group.example.name
}`
}

// CreateTestTerraformLocalsFile creates a Terraform file with locals
func CreateTestTerraformLocalsFile() string {
	return `# Locals-focused Terraform configuration
locals {
  environment = "production"
  project     = "example-project"
  
  common_tags = {
    Environment = local.environment
    Project     = local.project
    ManagedBy   = "terraform"
  }
  
  vpc_cidr = "10.0.0.0/16"
  
  availability_zones = [
    "us-west-2a",
    "us-west-2b", 
    "us-west-2c"
  ]
  
  private_subnets = [
    for i, az in local.availability_zones : 
    cidrsubnet(local.vpc_cidr, 8, i + 1)
  ]
  
  public_subnets = [
    for i, az in local.availability_zones : 
    cidrsubnet(local.vpc_cidr, 8, i + 101)
  ]
}`
}
