# ET-001 Complete Infrastructure Test Fixture

This directory contains a complete Terraform infrastructure configuration that satisfies ALL acceptance criteria for ET-001 (Infrastructure Security Configuration Evidence).

## ET-001 Requirements Coverage

### Network Security Groups (CC6.6, CC7.1)
- ✅ Security groups follow default-deny, explicit-allow principle
- ✅ No overly permissive security rules (0.0.0.0/0 on sensitive ports)
- ✅ Security group rules limiting access to specific ports and IP ranges

### Encryption at Rest (CC6.8)
- ✅ All production databases encrypted at rest with AES-256
- ✅ KMS key management evidence
- ✅ S3 buckets with encryption enabled

### Encryption in Transit (CC6.8)
- ✅ All API and web traffic uses TLS 1.2 or higher
- ✅ ACM certificates for HTTPS
- ✅ Load balancer HTTPS listeners with secure policies

### IAM Configuration (CC6.1, CC6.3)
- ✅ IAM roles have minimal necessary permissions documented
- ✅ Principle of least privilege applied
- ✅ Proper segregation of duties

### Multi-AZ Deployment (SO2, Resilience)
- ✅ Production services deployed in at least 2 availability zones
- ✅ RDS clusters with Multi-AZ configuration
- ✅ Auto Scaling Groups across multiple AZs
- ✅ Load balancers spanning multiple AZs

### Infrastructure as Code
- ✅ Infrastructure defined in version-controlled Terraform code
- ✅ All resources properly tagged
- ✅ Configuration follows best practices

## Expected Index Output

When this fixture is indexed, it should produce:

- **Total Resources:** ~50+
- **Control Coverage:**
  - CC6.1: ~15 resources (IAM roles, policies)
  - CC6.6: ~12 resources (security groups, network ACLs)
  - CC6.8: ~20 resources (KMS, S3 encryption, RDS encryption, TLS)
  - CC7.2: ~8 resources (CloudTrail, CloudWatch, logs)
  - SO2: ~6 resources (Auto Scaling, multi-AZ resources)

- **Security Attributes:**
  - encryption: ~20 resources
  - access_control: ~15 resources
  - network_security: ~12 resources
  - monitoring: ~8 resources
  - high_availability: ~6 resources

- **Compliance Status:** COMPLIANT (no high-severity findings)
- **Risk Distribution:**
  - High: 0
  - Medium: 0-2 (minor optimizations possible)
  - Low: 3-5 (informational)

## Testing Usage

```bash
# Build index from this fixture
grctool tool terraform-index build --path test/fixtures/terraform/et001_complete

# Query for ET-001 evidence
grctool tool terraform-index query --evidence-task ET-001 --output json

# Validate compliance
grctool tool terraform-security-analyzer \
  --security_domain all \
  --evidence_tasks ET-001 \
  --include_compliance_gaps true
```

## File Structure

- `main.tf` - Core infrastructure (VPC, networking, multi-AZ setup)
- `encryption.tf` - All encryption configurations (KMS, S3, RDS, TLS)
- `iam.tf` - Access control with least privilege
- `monitoring.tf` - Audit logging and monitoring
- `variables.tf` - Input variables
- `outputs.tf` - Output values
- `expected_index.json` - Expected index structure for validation
