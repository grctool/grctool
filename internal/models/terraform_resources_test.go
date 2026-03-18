package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformModule_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	mod := TerraformModule{
		FilePath: "/infra/main.tf",
		Resources: []TerraformResource{
			{
				Type:     "aws_instance",
				Name:     "web",
				Config:   map[string]interface{}{"instance_type": "t3.micro"},
				FilePath: "/infra/main.tf",
			},
		},
		Variables: []TerraformVariable{
			{
				Name:        "region",
				Type:        "string",
				Description: "AWS region",
				Default:     "us-east-1",
				Sensitive:   false,
			},
		},
		Outputs: []TerraformOutput{
			{
				Name:        "instance_id",
				Description: "EC2 instance ID",
			},
		},
		Locals: []TerraformLocal{
			{Name: "env", FilePath: "/infra/main.tf"},
		},
		ModuleCalls: []TerraformModuleCall{
			{
				Name:    "vpc",
				Source:  "terraform-aws-modules/vpc/aws",
				Version: "3.0.0",
			},
		},
		ProviderConfigs: []TerraformProvider{
			{
				Name:    "aws",
				Version: "~> 5.0",
				Config:  map[string]interface{}{"region": "us-east-1"},
			},
		},
		ParsedAt:          now,
		SecurityRelevance: []string{"CC6.1"},
		Metadata:          map[string]interface{}{"tool": "grctool"},
	}

	data, err := json.Marshal(mod)
	require.NoError(t, err)

	var decoded TerraformModule
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, mod.FilePath, decoded.FilePath)
	assert.Len(t, decoded.Resources, 1)
	assert.Len(t, decoded.Variables, 1)
	assert.Len(t, decoded.Outputs, 1)
	assert.Len(t, decoded.Locals, 1)
	assert.Len(t, decoded.ModuleCalls, 1)
	assert.Len(t, decoded.ProviderConfigs, 1)
}

func TestTerraformResource_WithLifecycle(t *testing.T) {
	t.Parallel()
	resource := TerraformResource{
		Type:      "aws_s3_bucket",
		Name:      "data",
		Config:    map[string]interface{}{"bucket": "my-data-bucket"},
		DependsOn: []string{"aws_kms_key.bucket_key"},
		Provider:  "aws.us-east",
		Lifecycle: &TerraformLifecycle{
			PreventDestroy:      true,
			CreateBeforeDestroy: false,
			IgnoreChanges:       []string{"tags"},
		},
		Tags:              map[string]interface{}{"env": "prod"},
		FilePath:          "/infra/storage.tf",
		SecurityRelevance: []string{"CC6.6", "CC6.7"},
	}

	data, err := json.Marshal(resource)
	require.NoError(t, err)

	var decoded TerraformResource
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "aws_s3_bucket", decoded.Type)
	assert.NotNil(t, decoded.Lifecycle)
	assert.True(t, decoded.Lifecycle.PreventDestroy)
	assert.Len(t, decoded.Lifecycle.IgnoreChanges, 1)
	assert.Len(t, decoded.DependsOn, 1)
}

func TestTerraformExpression_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	expr := TerraformExpression{
		Raw:       "var.region",
		Variables: []string{"var.region"},
		References: []TerraformReference{
			{Type: "var", Name: "region"},
		},
		IsConstant:   false,
		HasSensitive: false,
		CTYType:      "string",
	}

	data, err := json.Marshal(expr)
	require.NoError(t, err)

	var decoded TerraformExpression
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "var.region", decoded.Raw)
	assert.Len(t, decoded.Variables, 1)
	assert.Len(t, decoded.References, 1)
	assert.Equal(t, "var", decoded.References[0].Type)
}

func TestTerraformParseResult_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	result := TerraformParseResult{
		Modules: []TerraformModule{
			{FilePath: "/main.tf"},
		},
		Dependencies: []ResourceDependency{
			{
				From:           ResourceIdentifier{Type: "resource", Kind: "aws_instance", Name: "web"},
				To:             ResourceIdentifier{Type: "resource", Kind: "aws_security_group", Name: "web_sg"},
				DependencyType: "implicit",
				Attribute:      "vpc_security_group_ids",
			},
		},
		SecurityFindings: []SecurityFinding{
			{
				FindingType: "encryption",
				Severity:    "high",
				Title:       "Unencrypted S3 bucket",
			},
		},
		ParseSummary: &ParseSummary{
			FilesProcessed: 5,
			TotalResources: 20,
			ResourceCounts: map[string]int{"aws_instance": 3, "aws_s3_bucket": 2},
			ErrorCount:     0,
		},
		ParsedAt:    now,
		ToolVersion: "1.0.0",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded TerraformParseResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Modules, 1)
	assert.Len(t, decoded.Dependencies, 1)
	assert.Equal(t, "implicit", decoded.Dependencies[0].DependencyType)
	assert.Len(t, decoded.SecurityFindings, 1)
	assert.NotNil(t, decoded.ParseSummary)
	assert.Equal(t, 5, decoded.ParseSummary.FilesProcessed)
}

func TestNetworkTopology_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	network := NetworkTopology{
		VPCId: "vpc-123",
		CIDR:  "10.0.0.0/16",
		Subnets: []SubnetInfo{
			{Name: "public-1a", CIDR: "10.0.1.0/24", AvailabilityZone: "us-east-1a", Type: "public", IsPublic: true},
			{Name: "private-1a", CIDR: "10.0.2.0/24", AvailabilityZone: "us-east-1a", Type: "private", IsPublic: false},
		},
		InternetGateways: []string{"igw-123"},
		NATGateways:      []string{"nat-123"},
	}

	data, err := json.Marshal(network)
	require.NoError(t, err)

	var decoded NetworkTopology
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "vpc-123", decoded.VPCId)
	assert.Len(t, decoded.Subnets, 2)
	assert.True(t, decoded.Subnets[0].IsPublic)
	assert.False(t, decoded.Subnets[1].IsPublic)
}

func TestSecurityGroupInfo_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	sg := SecurityGroupInfo{
		Name:        "web-sg",
		Description: "Web server security group",
		VPC:         "vpc-123",
		IngressRules: []SecurityRule{
			{Protocol: "tcp", FromPort: "443", ToPort: "443", CIDRBlocks: []string{"0.0.0.0/0"}, Description: "HTTPS"},
		},
		EgressRules: []SecurityRule{
			{Protocol: "-1", CIDRBlocks: []string{"0.0.0.0/0"}, Description: "All outbound"},
		},
	}

	data, err := json.Marshal(sg)
	require.NoError(t, err)

	var decoded SecurityGroupInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "web-sg", decoded.Name)
	assert.Len(t, decoded.IngressRules, 1)
	assert.Equal(t, "443", decoded.IngressRules[0].FromPort)
}

func TestIAMResourceInfo_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	iam := IAMResourceInfo{
		Name:     "app-role",
		Type:     "role",
		Policies: []string{"arn:aws:iam::policy/ReadOnly"},
		AssumeRole: &AssumeRoleInfo{
			Principals: []string{"ec2.amazonaws.com"},
			Actions:    []string{"sts:AssumeRole"},
		},
		IsService: true,
	}

	data, err := json.Marshal(iam)
	require.NoError(t, err)

	var decoded IAMResourceInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "app-role", decoded.Name)
	assert.True(t, decoded.IsService)
	assert.NotNil(t, decoded.AssumeRole)
	assert.Len(t, decoded.AssumeRole.Principals, 1)
}

func TestHCLRange_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	r := HCLRange{
		Filename: "main.tf",
		Start:    HCLPos{Line: 10, Column: 1, Byte: 100},
		End:      HCLPos{Line: 20, Column: 3, Byte: 250},
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)

	var decoded HCLRange
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "main.tf", decoded.Filename)
	assert.Equal(t, 10, decoded.Start.Line)
	assert.Equal(t, 20, decoded.End.Line)
}

func TestTerraformVariable_WithValidation(t *testing.T) {
	t.Parallel()
	v := TerraformVariable{
		Name:        "instance_type",
		Type:        "string",
		Description: "EC2 instance type",
		Default:     "t3.micro",
		Sensitive:   false,
		Validation: []TerraformValidation{
			{
				Condition:    &TerraformExpression{Raw: "contains([\"t3.micro\", \"t3.small\"], var.instance_type)"},
				ErrorMessage: "Invalid instance type",
			},
		},
	}

	data, err := json.Marshal(v)
	require.NoError(t, err)

	var decoded TerraformVariable
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "instance_type", decoded.Name)
	assert.Len(t, decoded.Validation, 1)
	assert.Equal(t, "Invalid instance type", decoded.Validation[0].ErrorMessage)
}

func TestTerraformProvisioner_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	prov := TerraformProvisioner{
		Type:      "remote-exec",
		Config:    map[string]interface{}{"inline": []string{"echo hello"}},
		When:      "create",
		OnFailure: "fail",
		Connection: &TerraformConnection{
			Type:   "ssh",
			Config: map[string]interface{}{"host": "10.0.0.1"},
		},
	}

	data, err := json.Marshal(prov)
	require.NoError(t, err)

	var decoded TerraformProvisioner
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "remote-exec", decoded.Type)
	assert.NotNil(t, decoded.Connection)
	assert.Equal(t, "ssh", decoded.Connection.Type)
}
