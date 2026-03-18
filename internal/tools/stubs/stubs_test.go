package stubs_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/stubs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// StubGitHubAPIClient
// ---------------------------------------------------------------------------

func TestNewStubGitHubAPIClient(t *testing.T) {
	t.Parallel()
	s := stubs.NewStubGitHubAPIClient()
	require.NotNil(t, s)
	assert.NotNil(t, s.Repositories)
	assert.NotNil(t, s.Collaborators)
	assert.NotNil(t, s.Teams)
	assert.NotNil(t, s.Branches)
	assert.NotNil(t, s.Environments)
	assert.NotNil(t, s.SecuritySettings)
	assert.NotNil(t, s.OrgMembers)
	assert.NotNil(t, s.Errors)
}

func TestStubGitHubAPIClient_GetRepositoryCollaborators(t *testing.T) {
	t.Parallel()

	t.Run("returns configured collaborators", func(t *testing.T) {
		t.Parallel()
		s := stubs.NewStubGitHubAPIClient().
			WithCollaborators("org", "repo", stubs.CreateTestCollaborators())

		collabs, err := s.GetRepositoryCollaborators(context.Background(), "org", "repo")
		require.NoError(t, err)
		assert.Len(t, collabs, 3)
		assert.Equal(t, "admin-user", collabs[0].Login)
	})

	t.Run("returns empty for unknown repo", func(t *testing.T) {
		t.Parallel()
		s := stubs.NewStubGitHubAPIClient()
		collabs, err := s.GetRepositoryCollaborators(context.Background(), "org", "unknown")
		require.NoError(t, err)
		assert.Empty(t, collabs)
	})

	t.Run("returns configured error", func(t *testing.T) {
		t.Parallel()
		s := stubs.NewStubGitHubAPIClient().
			WithError("org/repo", fmt.Errorf("forbidden"))

		_, err := s.GetRepositoryCollaborators(context.Background(), "org", "repo")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})
}

func TestStubGitHubAPIClient_GetRepositoryTeams(t *testing.T) {
	t.Parallel()

	s := stubs.NewStubGitHubAPIClient().
		WithTeams("org", "repo", stubs.CreateTestTeams())

	teams, err := s.GetRepositoryTeams(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Equal(t, "admin-team", teams[0].Name)
	assert.Len(t, teams[0].Members, 1)
}

func TestStubGitHubAPIClient_GetRepositoryBranches(t *testing.T) {
	t.Parallel()

	s := stubs.NewStubGitHubAPIClient().
		WithBranches("org", "repo", stubs.CreateTestBranches())

	branches, err := s.GetRepositoryBranches(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.True(t, branches[0].Protected)
	require.NotNil(t, branches[0].Protection)
	assert.True(t, branches[0].Protection.Enabled)
}

func TestStubGitHubAPIClient_GetDeploymentEnvironments(t *testing.T) {
	t.Parallel()

	s := stubs.NewStubGitHubAPIClient().
		WithEnvironments("org", "repo", stubs.CreateTestEnvironments())

	envs, err := s.GetDeploymentEnvironments(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.Len(t, envs, 2)
	assert.Equal(t, "production", envs[0].Name)
	assert.Len(t, envs[0].ProtectionRules, 2)
}

func TestStubGitHubAPIClient_GetRepositorySecurity(t *testing.T) {
	t.Parallel()

	s := stubs.NewStubGitHubAPIClient().
		WithSecuritySettings("org", "repo", stubs.CreateTestSecuritySettings())

	settings, err := s.GetRepositorySecurity(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.True(t, settings.VulnerabilityAlertsEnabled)
	assert.True(t, settings.AutomatedSecurityFixesEnabled)
	assert.True(t, settings.SecretScanningEnabled)
	assert.False(t, settings.CodeScanningEnabled)
}

func TestStubGitHubAPIClient_GetOrganizationMembers(t *testing.T) {
	t.Parallel()

	members := []models.GitHubOrgMember{{Login: "member1", ID: 1}}
	s := stubs.NewStubGitHubAPIClient().
		WithOrgMembers("myorg", members)

	result, err := s.GetOrganizationMembers(context.Background(), "myorg")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "member1", result[0].Login)
}

func TestStubGitHubAPIClient_ChainedSetup(t *testing.T) {
	t.Parallel()

	repo := stubs.CreateTestRepository()
	s := stubs.NewStubGitHubAPIClient().
		WithRepository("org", "repo", repo).
		WithCollaborators("org", "repo", stubs.CreateTestCollaborators()).
		WithTeams("org", "repo", stubs.CreateTestTeams()).
		WithBranches("org", "repo", stubs.CreateTestBranches()).
		WithEnvironments("org", "repo", stubs.CreateTestEnvironments()).
		WithSecuritySettings("org", "repo", stubs.CreateTestSecuritySettings())

	assert.NotNil(t, s.Repositories["org/repo"])
	assert.Len(t, s.Collaborators["org/repo"], 3)
	assert.Len(t, s.Teams["org/repo"], 2)
}

// ---------------------------------------------------------------------------
// Test data factories
// ---------------------------------------------------------------------------

func TestCreateTestRepository(t *testing.T) {
	t.Parallel()
	repo := stubs.CreateTestRepository()
	assert.Equal(t, "test-repo", repo.Name)
	assert.Equal(t, "test-org/test-repo", repo.FullName)
	assert.True(t, repo.Private)
}

func TestCreateTestCollaborators(t *testing.T) {
	t.Parallel()
	collabs := stubs.CreateTestCollaborators()
	require.Len(t, collabs, 3)
	assert.True(t, collabs[0].Permissions.Admin)
	assert.False(t, collabs[1].Permissions.Admin)
	assert.True(t, collabs[1].Permissions.Push)
	assert.False(t, collabs[2].Permissions.Push)
}

func TestCreateTestSecuritySettings(t *testing.T) {
	t.Parallel()
	settings := stubs.CreateTestSecuritySettings()
	assert.True(t, settings.VulnerabilityAlertsEnabled)
	assert.False(t, settings.CodeScanningEnabled)
}

// ---------------------------------------------------------------------------
// StubFileReader
// ---------------------------------------------------------------------------

func TestNewStubFileReader(t *testing.T) {
	t.Parallel()
	fr := stubs.NewStubFileReader()
	require.NotNil(t, fr)
	assert.NotNil(t, fr.Files)
}

func TestStubFileReader_Open(t *testing.T) {
	t.Parallel()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		fr := stubs.NewStubFileReader().
			WithFile("/tmp/test.tf", stubs.CreateTestTerraformFile())

		rc, err := fr.Open("/tmp/test.tf")
		require.NoError(t, err)
		defer rc.Close()

		buf := make([]byte, 100)
		n, _ := rc.Read(buf)
		assert.Contains(t, string(buf[:n]), "Test Terraform")
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		fr := stubs.NewStubFileReader()
		_, err := fr.Open("/tmp/missing.tf")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("error configured", func(t *testing.T) {
		t.Parallel()
		fr := stubs.NewStubFileReader().
			WithError("/tmp/broken.tf", fmt.Errorf("permission denied"))

		_, err := fr.Open("/tmp/broken.tf")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})
}

func TestStubFileReader_Walk(t *testing.T) {
	t.Parallel()

	fr := stubs.NewStubFileReader().
		WithFile("/root/a.tf", "a").
		WithFile("/root/b.tf", "b").
		WithFile("/other/c.tf", "c")

	var visited []string
	err := fr.Walk("/root/", func(path string, info stubs.FileInfo, err error) error {
		visited = append(visited, path)
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, visited, 2)
}

func TestStubFileReader_Glob(t *testing.T) {
	t.Parallel()

	fr := stubs.NewStubFileReader().
		WithFile("/root/main.tf", "content").
		WithFile("/root/vars.tf", "content").
		WithFile("/other/main.tf", "content")

	matches, err := fr.Glob("/root/*")
	require.NoError(t, err)
	assert.Len(t, matches, 2)
}

func TestStubFileInfo(t *testing.T) {
	t.Parallel()

	info := &stubs.StubFileInfo{}
	// Exercise the interface methods - they return zero values by default
	assert.Equal(t, "", info.Name())
	assert.Equal(t, int64(0), info.Size())
	assert.False(t, info.IsDir())
	assert.NotNil(t, info.ModTime())
}

func TestStubReadCloser_Close(t *testing.T) {
	t.Parallel()
	fr := stubs.NewStubFileReader().WithFile("/tmp/test", "data")
	rc, err := fr.Open("/tmp/test")
	require.NoError(t, err)
	assert.NoError(t, rc.Close())
}

// ---------------------------------------------------------------------------
// Terraform test data factories
// ---------------------------------------------------------------------------

func TestCreateTestTerraformFile(t *testing.T) {
	t.Parallel()
	content := stubs.CreateTestTerraformFile()
	assert.Contains(t, content, "aws_s3_bucket")
	assert.Contains(t, content, "aws_iam_role")
	assert.Contains(t, content, "aws_security_group")
}

func TestCreateTestTerraformSecurityFile(t *testing.T) {
	t.Parallel()
	content := stubs.CreateTestTerraformSecurityFile()
	assert.Contains(t, content, "aws_kms_key")
	assert.Contains(t, content, "aws_cloudtrail")
}

func TestCreateTestTerraformModuleFile(t *testing.T) {
	t.Parallel()
	content := stubs.CreateTestTerraformModuleFile()
	assert.Contains(t, content, "module \"networking\"")
	assert.Contains(t, content, "module \"database\"")
}

func TestCreateTestTerraformLocalsFile(t *testing.T) {
	t.Parallel()
	content := stubs.CreateTestTerraformLocalsFile()
	assert.Contains(t, content, "locals {")
	assert.Contains(t, content, "common_tags")
}
