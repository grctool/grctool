package types_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Response constructors
// ---------------------------------------------------------------------------

func TestNewSuccessResponse(t *testing.T) {
	t.Parallel()

	t.Run("with metadata", func(t *testing.T) {
		t.Parallel()
		meta := map[string]interface{}{"key": "value"}
		resp := types.NewSuccessResponse("test-tool", "hello", nil, meta)

		assert.True(t, resp.Success)
		assert.Equal(t, "test-tool", resp.ToolName)
		assert.Equal(t, "hello", resp.Content)
		assert.Equal(t, "", resp.Error)
		assert.Equal(t, "value", resp.Metadata["key"])
		assert.False(t, resp.ExecutedAt.IsZero())
	})

	t.Run("nil metadata becomes empty map", func(t *testing.T) {
		t.Parallel()
		resp := types.NewSuccessResponse("t", "c", nil, nil)
		require.NotNil(t, resp.Metadata)
		assert.Empty(t, resp.Metadata)
	})

	t.Run("with evidence source", func(t *testing.T) {
		t.Parallel()
		src := &models.EvidenceSource{Type: "github", Resource: "repo"}
		resp := types.NewSuccessResponse("t", "c", src, nil)
		require.NotNil(t, resp.Source)
		assert.Equal(t, "github", resp.Source.Type)
	})
}

func TestNewErrorResponse(t *testing.T) {
	t.Parallel()

	resp := types.NewErrorResponse("bad-tool", "something broke", nil)

	assert.False(t, resp.Success)
	assert.Equal(t, "bad-tool", resp.ToolName)
	assert.Equal(t, "something broke", resp.Error)
	assert.Equal(t, "", resp.Content)
	require.NotNil(t, resp.Metadata)
}

// ---------------------------------------------------------------------------
// ToolResponse implements Response interface
// ---------------------------------------------------------------------------

func TestToolResponseImplementsResponse(t *testing.T) {
	t.Parallel()
	resp := types.NewSuccessResponse("t", "content", nil, map[string]interface{}{"k": 1})
	var r types.Response = resp
	assert.Equal(t, "content", r.GetContent())
	assert.Equal(t, 1, r.GetMetadata()["k"])
}

// ---------------------------------------------------------------------------
// ToolResponse JSON round-trip
// ---------------------------------------------------------------------------

func TestToolResponseJSONRoundTrip(t *testing.T) {
	t.Parallel()
	orig := types.NewSuccessResponse("my-tool", "data", nil, map[string]interface{}{"count": float64(42)})
	orig.ExecutionID = "exec-123"
	orig.RequestType = "scan"

	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded types.ToolResponse
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, orig.Content, decoded.Content)
	assert.Equal(t, orig.ToolName, decoded.ToolName)
	assert.Equal(t, orig.Success, decoded.Success)
	assert.Equal(t, orig.ExecutionID, decoded.ExecutionID)
	assert.Equal(t, orig.RequestType, decoded.RequestType)
	assert.Equal(t, float64(42), decoded.Metadata["count"])
}

// ---------------------------------------------------------------------------
// Request validation
// ---------------------------------------------------------------------------

func TestGitHubRequestValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		req := &types.GitHubRequest{Query: "security", Repository: "org/repo"}
		err := req.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "issues", req.SearchType) // default
		assert.Equal(t, 50, req.MaxResults)        // default
	})

	t.Run("missing required query", func(t *testing.T) {
		t.Parallel()
		req := &types.GitHubRequest{}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "github request validation failed")
	})

	t.Run("invalid search type", func(t *testing.T) {
		t.Parallel()
		req := &types.GitHubRequest{Query: "test", SearchType: "invalid"}
		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestEvidenceTaskRequestValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		req := &types.EvidenceTaskRequest{TaskRef: "ET-0001"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty task ref", func(t *testing.T) {
		t.Parallel()
		req := &types.EvidenceTaskRequest{}
		assert.Error(t, req.Validate())
	})
}

func TestPromptAssemblerRequestValidation(t *testing.T) {
	t.Parallel()

	req := &types.PromptAssemblerRequest{TaskRef: "ET-0001"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "markdown", req.Format) // default

	req2 := &types.PromptAssemblerRequest{TaskRef: "ET-0001", Format: "text"}
	require.NoError(t, req2.Validate())
	assert.Equal(t, "text", req2.Format)
}

func TestDocsReaderRequestValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		req := &types.DocsReaderRequest{QueryTerms: []string{"access control"}}
		require.NoError(t, req.Validate())
		assert.Equal(t, 10, req.MaxResults) // default
	})

	t.Run("missing query terms", func(t *testing.T) {
		t.Parallel()
		req := &types.DocsReaderRequest{}
		assert.Error(t, req.Validate())
	})
}

func TestGoogleWorkspaceRequestValidation(t *testing.T) {
	t.Parallel()

	req := &types.GoogleWorkspaceRequest{Query: "security policy"}
	require.NoError(t, req.Validate())
	assert.Equal(t, 25, req.MaxResults)
}

func TestControlSummaryRequestValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		req := &types.ControlSummaryRequest{TaskRef: "ET-0001"}
		assert.NoError(t, req.Validate())
	})

	t.Run("missing task ref", func(t *testing.T) {
		t.Parallel()
		req := &types.ControlSummaryRequest{}
		assert.Error(t, req.Validate())
	})
}

func TestPolicySummaryRequestValidation(t *testing.T) {
	t.Parallel()

	req := &types.PolicySummaryRequest{TaskRef: "ET-0001"}
	require.NoError(t, req.Validate())
	assert.Equal(t, 1000, req.MaxLength) // default
}

func TestTerraformRequestValidation(t *testing.T) {
	t.Parallel()

	t.Run("defaults applied", func(t *testing.T) {
		t.Parallel()
		req := &types.TerraformRequest{}
		err := req.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "csv", req.OutputFormat)
	})

	t.Run("invalid output format", func(t *testing.T) {
		t.Parallel()
		req := &types.TerraformRequest{OutputFormat: "xml"}
		assert.Error(t, req.Validate())
	})
}

// ---------------------------------------------------------------------------
// Conversion functions
// ---------------------------------------------------------------------------

func TestConvertLegacyParamsToRequest(t *testing.T) {
	t.Parallel()

	params := map[string]interface{}{
		"query":      "security",
		"repository": "org/repo",
		"max_results": float64(10),
	}

	req := &types.GitHubRequest{}
	err := types.ConvertLegacyParamsToRequest(params, req)
	require.NoError(t, err)
	assert.Equal(t, "security", req.Query)
	assert.Equal(t, "org/repo", req.Repository)
	assert.Equal(t, 10, req.MaxResults)
}

func TestConvertRequestToLegacyParams(t *testing.T) {
	t.Parallel()

	req := &types.GitHubRequest{Query: "test", Repository: "a/b", MaxResults: 20, SearchType: "issues"}
	params, err := types.ConvertRequestToLegacyParams(req)
	require.NoError(t, err)
	assert.Equal(t, "test", params["query"])
	assert.Equal(t, "a/b", params["repository"])
}

func TestValidateAndConvertParams(t *testing.T) {
	t.Parallel()

	t.Run("registered tool", func(t *testing.T) {
		t.Parallel()
		params := map[string]interface{}{"query": "vuln", "repository": "org/repo"}
		req, err := types.ValidateAndConvertParams("github_searcher", params)
		require.NoError(t, err)
		ghReq, ok := req.(*types.GitHubRequest)
		require.True(t, ok)
		assert.Equal(t, "vuln", ghReq.Query)
	})

	t.Run("unregistered tool", func(t *testing.T) {
		t.Parallel()
		_, err := types.ValidateAndConvertParams("nonexistent_tool", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have a registered request type")
	})

	t.Run("validation failure", func(t *testing.T) {
		t.Parallel()
		// github_searcher requires query
		_, err := types.ValidateAndConvertParams("github_searcher", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

func TestExtractRequestFromInterface(t *testing.T) {
	t.Parallel()

	t.Run("from map", func(t *testing.T) {
		t.Parallel()
		input := map[string]interface{}{"task_ref": "ET-0001"}
		req := &types.EvidenceTaskRequest{}
		require.NoError(t, types.ExtractRequestFromInterface(input, req))
		assert.Equal(t, "ET-0001", req.TaskRef)
	})

	t.Run("from JSON string", func(t *testing.T) {
		t.Parallel()
		input := `{"task_ref": "ET-0002"}`
		req := &types.EvidenceTaskRequest{}
		require.NoError(t, types.ExtractRequestFromInterface(input, req))
		assert.Equal(t, "ET-0002", req.TaskRef)
	})

	t.Run("from struct", func(t *testing.T) {
		t.Parallel()
		input := struct {
			TaskRef string `json:"task_ref"`
		}{TaskRef: "ET-0003"}
		req := &types.EvidenceTaskRequest{}
		require.NoError(t, types.ExtractRequestFromInterface(input, req))
		assert.Equal(t, "ET-0003", req.TaskRef)
	})

	t.Run("invalid JSON string", func(t *testing.T) {
		t.Parallel()
		req := &types.EvidenceTaskRequest{}
		err := types.ExtractRequestFromInterface("not json", req)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RequestMatcher
// ---------------------------------------------------------------------------

func TestRequestMatcher(t *testing.T) {
	t.Parallel()

	t.Run("create for registered tool", func(t *testing.T) {
		t.Parallel()
		req, err := types.DefaultRequestMatcher.CreateRequestForTool("github_searcher")
		require.NoError(t, err)
		_, ok := req.(*types.GitHubRequest)
		assert.True(t, ok)
	})

	t.Run("create for unregistered tool", func(t *testing.T) {
		t.Parallel()
		_, err := types.DefaultRequestMatcher.CreateRequestForTool("unknown_tool")
		assert.Error(t, err)
	})

	t.Run("custom matcher", func(t *testing.T) {
		t.Parallel()
		m := types.NewRequestMatcher()
		m.RegisterRequestType("custom", func() types.Request { return &types.EvidenceTaskRequest{} })
		req, err := m.CreateRequestForTool("custom")
		require.NoError(t, err)
		_, ok := req.(*types.EvidenceTaskRequest)
		assert.True(t, ok)
	})
}

func TestListRegisteredRequestTypes(t *testing.T) {
	t.Parallel()
	registered := types.ListRegisteredRequestTypes()
	assert.Contains(t, registered, "github_searcher")
	assert.Contains(t, registered, "terraform_scanner")
	assert.Contains(t, registered, "evidence_task_details")
}

func TestGetRequestTypeForTool(t *testing.T) {
	t.Parallel()

	rt, err := types.GetRequestTypeForTool("github_searcher")
	require.NoError(t, err)
	assert.Equal(t, "GitHubRequest", rt.Name())
}

func TestGetRequestFieldInfo(t *testing.T) {
	t.Parallel()

	fields, err := types.GetRequestFieldInfo("github_searcher")
	require.NoError(t, err)

	queryField, ok := fields["query"]
	require.True(t, ok)
	assert.True(t, queryField.Required)
	assert.Equal(t, "string", queryField.Type)

	repoField, ok := fields["repository"]
	require.True(t, ok)
	assert.False(t, repoField.Required)
}

// ---------------------------------------------------------------------------
// ToolAdapter
// ---------------------------------------------------------------------------

// fakeLegacyTool implements types.LegacyTool for testing the adapter.
type fakeLegacyTool struct {
	name string
}

func (f *fakeLegacyTool) Name() string        { return f.name }
func (f *fakeLegacyTool) Description() string  { return "fake tool" }
func (f *fakeLegacyTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{Name: f.name, Description: "fake"}
}
func (f *fakeLegacyTool) Execute(_ context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return "executed", &models.EvidenceSource{Type: "test", Metadata: map[string]interface{}{"ok": true}}, nil
}

func TestToolAdapter(t *testing.T) {
	t.Parallel()

	legacy := &fakeLegacyTool{name: "my-tool"}
	adapter := types.NewToolAdapter(legacy)

	t.Run("Name delegates", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "my-tool", adapter.Name())
	})

	t.Run("Description delegates", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "fake tool", adapter.Description())
	})

	t.Run("GetClaudeToolDefinition delegates", func(t *testing.T) {
		t.Parallel()
		def := adapter.GetClaudeToolDefinition()
		assert.Equal(t, "my-tool", def.Name)
	})

	t.Run("Execute delegates to legacy", func(t *testing.T) {
		t.Parallel()
		content, src, err := adapter.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, "executed", content)
		assert.Equal(t, "test", src.Type)
	})

	t.Run("ExecuteTyped converts and delegates", func(t *testing.T) {
		t.Parallel()
		req := &types.EvidenceTaskRequest{TaskRef: "ET-0001"}
		resp, err := adapter.ExecuteTyped(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "executed", resp.GetContent())
	})
}

// ---------------------------------------------------------------------------
// ValidateStruct
// ---------------------------------------------------------------------------

func TestValidateStruct(t *testing.T) {
	t.Parallel()

	type sample struct {
		Name string `validate:"required"`
	}

	assert.NoError(t, types.ValidateStruct(&sample{Name: "ok"}))
	assert.Error(t, types.ValidateStruct(&sample{}))
}
