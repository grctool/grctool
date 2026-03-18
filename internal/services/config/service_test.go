package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// NewService
// ---------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log)
	require.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// GenerateDefaultConfig
// ---------------------------------------------------------------------------

func TestGenerateDefaultConfig(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	cfg := svc.GenerateDefaultConfig()
	require.NotNil(t, cfg)

	// Check top-level keys
	assert.Contains(t, cfg, "tugboat")
	assert.Contains(t, cfg, "storage")
	assert.Contains(t, cfg, "logging")
	assert.Contains(t, cfg, "evidence")

	// Check tugboat sub-keys
	tugboat, ok := cfg["tugboat"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://api-my.tugboatlogic.com", tugboat["base_url"])
	assert.Equal(t, "YOUR_ORG_ID", tugboat["org_id"])
	assert.Equal(t, "browser", tugboat["auth_mode"])
}

// ---------------------------------------------------------------------------
// GenerateConfigTemplate
// ---------------------------------------------------------------------------

func TestGenerateConfigTemplate_Minimal(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	cfg, err := svc.GenerateConfigTemplate("minimal")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Minimal config should have tugboat and storage only
	assert.Contains(t, cfg, "tugboat")
	assert.Contains(t, cfg, "storage")
	assert.NotContains(t, cfg, "logging")
}

func TestGenerateConfigTemplate_Extended(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	cfg, err := svc.GenerateConfigTemplate("extended")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Extended config should have evidence section with github and google_docs
	evidence, ok := cfg["evidence"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, evidence, "github")
	assert.Contains(t, evidence, "google_docs")
	assert.Contains(t, evidence, "terraform")
}

func TestGenerateConfigTemplate_Default(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	cfg, err := svc.GenerateConfigTemplate("default")
	require.NoError(t, err)

	// Default should be same as GenerateDefaultConfig
	defaultCfg := svc.GenerateDefaultConfig()
	assert.Equal(t, defaultCfg, cfg)
}

func TestGenerateConfigTemplate_UnknownType(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	cfg, err := svc.GenerateConfigTemplate("nonexistent")
	require.NoError(t, err)
	// Unknown types fall through to default
	assert.Contains(t, cfg, "tugboat")
}

// ---------------------------------------------------------------------------
// SaveConfigFile
// ---------------------------------------------------------------------------

func TestSaveConfigFile(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-config.yaml")

	cfg := svc.GenerateDefaultConfig()
	err := svc.SaveConfigFile(cfg, outputPath)
	require.NoError(t, err)

	// Verify file exists and is valid YAML
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var loaded map[string]interface{}
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Contains(t, loaded, "tugboat")
}

func TestSaveConfigFile_CreatesDirectory(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "subdir", "nested", "config.yaml")

	cfg := map[string]interface{}{"test": "value"}
	err := svc.SaveConfigFile(cfg, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// InitializeConfig
// ---------------------------------------------------------------------------

func TestInitializeConfig_NewFile(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".grctool.yaml")

	err := svc.InitializeConfig(outputPath, false)
	require.NoError(t, err)

	// File should exist
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tugboat")
}

func TestInitializeConfig_ExistingFileNoForce(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".grctool.yaml")

	// Create existing file
	err := os.WriteFile(outputPath, []byte("existing"), 0644)
	require.NoError(t, err)

	// Should fail without force
	err = svc.InitializeConfig(outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInitializeConfig_ExistingFileWithForce(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".grctool.yaml")

	// Create existing file
	err := os.WriteFile(outputPath, []byte("existing"), 0644)
	require.NoError(t, err)

	// Should succeed with force
	err = svc.InitializeConfig(outputPath, true)
	require.NoError(t, err)

	// File should be overwritten
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tugboat")
}

// ---------------------------------------------------------------------------
// ConfigValidator
// ---------------------------------------------------------------------------

func TestNewConfigValidator(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	v := NewConfigValidator(cfg)
	require.NotNil(t, v)
}

func TestValidate_AllChecksRun(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			BaseURL:  "https://api-my.tugboatlogic.com",
			AuthMode: "browser",
		},
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	v := NewConfigValidator(cfg)
	result, err := v.Validate(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)

	// All checks should be present
	assert.Contains(t, result.Checks, "paths")
	assert.Contains(t, result.Checks, "permissions")
	assert.Contains(t, result.Checks, "environment")
	assert.Contains(t, result.Checks, "tugboat_connectivity")
	assert.Contains(t, result.Checks, "tools")
	assert.Contains(t, result.Checks, "storage")

	// Duration should be positive
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

func TestValidate_ValidConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			BaseURL:  "https://api-my.tugboatlogic.com",
			AuthMode: "browser",
		},
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	v := NewConfigValidator(cfg)
	result, err := v.Validate(context.Background())
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidatePaths_EmptyRequired(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: "", // Empty required path
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidatePaths(context.Background(), result)

	check := result.Checks["paths"]
	assert.Equal(t, "fail", check.Status)
	assert.False(t, result.Valid)
}

func TestValidatePaths_ValidDirectory(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidatePaths(context.Background(), result)

	check := result.Checks["paths"]
	assert.Equal(t, "pass", check.Status)
	assert.True(t, result.Valid)
}

func TestValidatePaths_CreatesNonexistentDirectory(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new-data-dir")

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: newDir,
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidatePaths(context.Background(), result)

	assert.Equal(t, "pass", result.Checks["paths"].Status)
	// Directory should now exist
	_, err := os.Stat(newDir)
	assert.NoError(t, err)
}

func TestValidatePermissions_WritableDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidatePermissions(context.Background(), result)

	assert.Equal(t, "pass", result.Checks["permissions"].Status)
}

func TestValidateEnvironmentVariables_BrowserAuth(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			AuthMode: "browser",
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateEnvironmentVariables(context.Background(), result)

	// Browser auth mode should pass even without credentials
	assert.Equal(t, "pass", result.Checks["environment"].Status)
}

func TestValidateEnvironmentVariables_MissingCredentials(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			AuthMode:     "form", // Not browser
			CookieHeader: "",
			BearerToken:  "",
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateEnvironmentVariables(context.Background(), result)

	assert.Equal(t, "fail", result.Checks["environment"].Status)
	assert.False(t, result.Valid)
}

func TestValidateTugboatConnectivity_MissingURL(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			BaseURL: "",
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateTugboatConnectivity(context.Background(), result)

	assert.Equal(t, "fail", result.Checks["tugboat_connectivity"].Status)
	assert.False(t, result.Valid)
}

func TestValidateTugboatConnectivity_ValidURL(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			BaseURL: "https://api-my.tugboatlogic.com",
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateTugboatConnectivity(context.Background(), result)

	assert.Equal(t, "pass", result.Checks["tugboat_connectivity"].Status)
}

func TestValidateStorageConfiguration_EmptyDataDir(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: "",
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateStorageConfiguration(context.Background(), result)

	assert.Equal(t, "fail", result.Checks["storage"].Status)
	assert.False(t, result.Valid)
}

func TestValidateStorageConfiguration_ValidDataDir(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: "/some/path",
		},
	}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateStorageConfiguration(context.Background(), result)

	assert.Equal(t, "pass", result.Checks["storage"].Status)
}

func TestValidateToolConfigurations_AlwaysPass(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}

	v := NewConfigValidator(cfg).(*ConfigValidatorImpl)
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}
	v.ValidateToolConfigurations(context.Background(), result)

	assert.Equal(t, "pass", result.Checks["tools"].Status)
}

// ---------------------------------------------------------------------------
// RenderClaudeMd (template rendering)
// ---------------------------------------------------------------------------

func TestRenderClaudeMd(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir:  "/data/isms",
			CacheDir: "/data/isms/.cache",
			Paths: config.StoragePaths{
				Docs:     "docs",
				Evidence: "evidence",
				Prompts:  "prompts",
				Cache:    ".cache",
			},
		},
		Tugboat: config.TugboatConfig{
			OrgID: "test-org",
		},
	}

	content, err := RenderClaudeMd(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, content)

	// Check key template variables were substituted
	assert.Contains(t, content, "/data/isms")
	assert.Contains(t, content, "test-org")
	assert.Contains(t, content, "CLAUDE.md")
	assert.Contains(t, content, "GRCTool")
	assert.Contains(t, content, "PROJECT OVERVIEW")
	assert.Contains(t, content, "COMMON COMMANDS")
}

func TestRenderClaudeMd_EmptyConfig(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}

	content, err := RenderClaudeMd(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	// Should still render without errors even with empty config
	assert.Contains(t, content, "GRCTool")
}

// ---------------------------------------------------------------------------
// ClaudeMdData / ToolInfo types
// ---------------------------------------------------------------------------

func TestClaudeMdData_Fields(t *testing.T) {
	t.Parallel()
	data := ClaudeMdData{
		DataDir:      "/data",
		DocsPath:     "docs",
		EvidencePath: "evidence",
		OrgID:        "test",
		ToolCount:    5,
	}
	assert.Equal(t, "/data", data.DataDir)
	assert.Equal(t, 5, data.ToolCount)
}

func TestToolInfo_Fields(t *testing.T) {
	t.Parallel()
	ti := ToolInfo{Name: "github-permissions", Description: "Extract permissions"}
	assert.Equal(t, "github-permissions", ti.Name)
	assert.Equal(t, "Extract permissions", ti.Description)
}

// ---------------------------------------------------------------------------
// getShortDescription
// ---------------------------------------------------------------------------

func TestGetShortDescription_KnownTool(t *testing.T) {
	t.Parallel()
	// This should match the "terraform-security-indexer" keyword in description
	desc := getShortDescription("terraform-security-indexer based tool")
	assert.Equal(t, "Infrastructure security", desc)
}

func TestGetShortDescription_ShortDesc(t *testing.T) {
	t.Parallel()
	desc := getShortDescription("A short description")
	assert.Equal(t, "A short description", desc)
}

func TestGetShortDescription_LongDesc(t *testing.T) {
	t.Parallel()
	longDesc := "This is a very long description that exceeds fifty characters in total length"
	desc := getShortDescription(longDesc)
	assert.Len(t, desc, 50)
	assert.True(t, len(desc) <= 50)
}

// ---------------------------------------------------------------------------
// NormalizeLanguageName (conversion helper used in templates)
// ---------------------------------------------------------------------------

func TestValidationResult_Fields(t *testing.T) {
	t.Parallel()
	vr := ValidationResult{
		Valid:  true,
		Checks: map[string]ValidationCheck{},
		Errors: []string{},
	}
	assert.True(t, vr.Valid)
	assert.Empty(t, vr.Errors)
}

func TestValidationCheck_Fields(t *testing.T) {
	t.Parallel()
	vc := ValidationCheck{
		Name:    "test",
		Status:  "pass",
		Message: "all good",
	}
	assert.Equal(t, "pass", vc.Status)
}

// ---------------------------------------------------------------------------
// GenerateClaudeMd (file writing)
// ---------------------------------------------------------------------------

func TestGenerateClaudeMd_ExistingFileNoForce(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	svc := NewService(log).(*ServiceImpl)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Create existing file
	err := os.WriteFile(outputPath, []byte("existing content"), 0644)
	require.NoError(t, err)

	err = svc.GenerateClaudeMd(outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
