// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestSemanticSearch(t *testing.T, rootPath string) *SemanticSearch {
	t.Helper()
	return &SemanticSearch{
		rootPath: rootPath,
		logger:   testhelpers.NewStubLogger(),
	}
}

func newTestSemanticSearchTool(t *testing.T, rootPath string) *SemanticSearchTool {
	t.Helper()
	return &SemanticSearchTool{
		search: newTestSemanticSearch(t, rootPath),
		logger: testhelpers.NewStubLogger(),
	}
}

// ---------------------------------------------------------------------------
// tokenize
// ---------------------------------------------------------------------------

func TestSemanticSearch_Tokenize(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	tests := map[string]struct {
		input string
		want  int // minimum tokens expected
	}{
		"camelCase":            {input: "calculateRelevanceScore", want: 1},
		"snake_case":           {input: "calculate_relevance_score", want: 3},
		"simple word":          {input: "hello", want: 1},
		"empty string":         {input: "", want: 0},
		"all uppercase":        {input: "HTTP", want: 1},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tokens := ss.tokenize(tc.input)
			assert.GreaterOrEqual(t, len(tokens), tc.want, "for input %q got tokens: %v", tc.input, tokens)
		})
	}
}

// ---------------------------------------------------------------------------
// tokenizeSlice
// ---------------------------------------------------------------------------

func TestSemanticSearch_TokenizeSlice(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())
	tokens := ss.tokenizeSlice([]string{"calculateScore", "buildIndex"})
	assert.NotEmpty(t, tokens)
}

// ---------------------------------------------------------------------------
// dedupeStrings
// ---------------------------------------------------------------------------

func TestSemanticSearch_DedupeStrings(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	t.Run("removes duplicates", func(t *testing.T) {
		t.Parallel()
		result := ss.dedupeStrings([]string{"a", "b", "a", "c", "b"})
		assert.Len(t, result, 3)
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()
		result := ss.dedupeStrings(nil)
		assert.Empty(t, result)
	})
}

// ---------------------------------------------------------------------------
// truncateText
// ---------------------------------------------------------------------------

func TestSemanticSearch_TruncateText(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	t.Run("short text unchanged", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello", ss.truncateText("hello", 10))
	})

	t.Run("long text truncated", func(t *testing.T) {
		t.Parallel()
		result := ss.truncateText("this is a long text", 10)
		assert.LessOrEqual(t, len(result), 13) // 10 + "..."
		assert.True(t, len(result) <= 13)
	})
}

// ---------------------------------------------------------------------------
// termMatchScore
// ---------------------------------------------------------------------------

func TestSemanticSearch_TermMatchScore(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	t.Run("no tokens", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, ss.termMatchScore(nil, []string{"test"}))
	})

	t.Run("no query terms", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, ss.termMatchScore([]string{"token"}, nil))
	})

	t.Run("all terms match", func(t *testing.T) {
		t.Parallel()
		score := ss.termMatchScore([]string{"access", "control"}, []string{"access", "control"})
		assert.Equal(t, 1.0, score)
	})

	t.Run("partial match", func(t *testing.T) {
		t.Parallel()
		score := ss.termMatchScore([]string{"access", "other"}, []string{"access", "control"})
		assert.Greater(t, score, 0.0)
		assert.Less(t, score, 1.0)
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		score := ss.termMatchScore([]string{"nothing"}, []string{"something"})
		assert.Equal(t, 0.0, score)
	})
}

// ---------------------------------------------------------------------------
// classifyComment
// ---------------------------------------------------------------------------

func TestSemanticSearch_ClassifyComment(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	tests := map[string]struct {
		text string
		want string
	}{
		"TODO comment":    {text: "TODO: fix this later", want: "todo"},
		"FIXME comment":   {text: "FIXME: broken", want: "fixme"},
		"doc comment":     {text: "Package tools provides evidence collection capabilities for automated compliance frameworks and more stuff here", want: "doc"},
		"short comment":   {text: "short", want: "comment"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ss.classifyComment(tc.text)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// inferFilePurpose
// ---------------------------------------------------------------------------

func TestSemanticSearch_InferFilePurpose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ss := newTestSemanticSearch(t, dir)

	t.Run("test file", func(t *testing.T) {
		t.Parallel()
		purpose := ss.inferFilePurpose("validator_test.go", nil)
		assert.Contains(t, purpose, "test")
	})

	t.Run("main file", func(t *testing.T) {
		t.Parallel()
		purpose := ss.inferFilePurpose("main.go", nil)
		assert.Contains(t, purpose, "entry point")
	})
}

// ---------------------------------------------------------------------------
// inferFunctionPurpose
// ---------------------------------------------------------------------------

func TestSemanticSearch_InferFunctionPurpose(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	tests := map[string]struct {
		fn   FunctionInfo
		want string
	}{
		"New constructor": {
			fn:   FunctionInfo{Name: "NewValidator"},
			want: "constructor",
		},
		"Get getter": {
			fn:   FunctionInfo{Name: "GetPolicy"},
			want: "getter",
		},
		"Is predicate": {
			fn:   FunctionInfo{Name: "IsValid"},
			want: "predicate",
		},
		"Test function": {
			fn:   FunctionInfo{Name: "TestValidator"},
			want: "test",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			purpose := ss.inferFunctionPurpose(tc.fn)
			assert.Contains(t, purpose, tc.want)
		})
	}
}

// ---------------------------------------------------------------------------
// inferTypePurpose
// ---------------------------------------------------------------------------

func TestSemanticSearch_InferTypePurpose(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())

	t.Run("interface type", func(t *testing.T) {
		t.Parallel()
		purpose := ss.inferTypePurpose(TypeInfo{Name: "Tool", Kind: "interface"})
		assert.Contains(t, purpose, "interface")
	})

	t.Run("error type", func(t *testing.T) {
		t.Parallel()
		purpose := ss.inferTypePurpose(TypeInfo{Name: "ValidationError"})
		assert.Contains(t, purpose, "error")
	})
}

// ---------------------------------------------------------------------------
// extractFunctionKeywords, extractTypeKeywords, extractCommentKeywords
// ---------------------------------------------------------------------------

func TestSemanticSearch_ExtractFunctionKeywords(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())
	keywords := ss.extractFunctionKeywords(FunctionInfo{
		Name:       "ValidateTaskReference",
		Receiver:   "Validator",
		Parameters: []string{"string"},
		DocComment: "validates task references",
	})
	assert.NotEmpty(t, keywords)
}

func TestSemanticSearch_ExtractTypeKeywords(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())
	keywords := ss.extractTypeKeywords(TypeInfo{
		Name:   "ValidationResult",
		Fields: []string{"Valid", "Errors", "Warnings"},
	})
	assert.NotEmpty(t, keywords)
}

func TestSemanticSearch_ExtractCommentKeywords(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())
	keywords := ss.extractCommentKeywords("This validates evidence task references for SOC2 compliance")
	assert.NotEmpty(t, keywords)
}

// ---------------------------------------------------------------------------
// formatFunctionContext, formatTypeContext
// ---------------------------------------------------------------------------

func TestSemanticSearch_FormatFunctionContext(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())
	ctx := ss.formatFunctionContext(FunctionInfo{
		Name:        "Execute",
		File:        "validator.go",
		Line:        42,
		Receiver:    "Validator",
		Parameters:  []string{"ctx context.Context", "params map[string]interface{}"},
		ReturnTypes: []string{"string", "error"},
		DocComment:  "Execute runs the validation",
	})
	assert.Contains(t, ctx, "Execute")
	assert.Contains(t, ctx, "Validator")
}

func TestSemanticSearch_FormatTypeContext(t *testing.T) {
	t.Parallel()

	ss := newTestSemanticSearch(t, t.TempDir())
	ctx := ss.formatTypeContext(TypeInfo{
		Name:   "Registry",
		File:   "registry.go",
		Fields: []string{"tools", "mutex"},
	})
	assert.Contains(t, ctx, "Registry")
}

// ---------------------------------------------------------------------------
// BuildIndex + Search integration
// ---------------------------------------------------------------------------

func TestSemanticSearch_BuildIndexAndSearch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "tools", "validator.go", `
// Validator provides input validation and path safety checks
type Validator struct {
	dataDir string
}

// NewValidator creates a new validator with the specified data directory
func NewValidator(dataDir string) *Validator {
	return &Validator{dataDir: dataDir}
}

// ValidateTaskReference validates and normalizes task references
func (v *Validator) ValidateTaskReference(taskRef string) error {
	return nil
}
`)

	ss := newTestSemanticSearch(t, dir)

	// Build index
	err := ss.BuildIndex(context.Background())
	require.NoError(t, err)

	index := ss.GetIndex()
	require.NotNil(t, index)
	assert.NotEmpty(t, index.Functions)
	assert.NotEmpty(t, index.Types)

	// Search
	results, err := ss.Search(context.Background(), SearchQuery{
		Query:      "validator task reference",
		Limit: 10,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

// ---------------------------------------------------------------------------
// SaveIndex / LoadIndex
// ---------------------------------------------------------------------------

func TestSemanticSearch_SaveAndLoadIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "main", "main.go", `
func hello() string { return "hello" }
`)

	ss := newTestSemanticSearch(t, dir)
	require.NoError(t, ss.BuildIndex(context.Background()))

	indexPath := filepath.Join(dir, "index.json")

	// Save
	err := ss.SaveIndex(indexPath)
	require.NoError(t, err)

	// Load into new instance
	ss2 := newTestSemanticSearch(t, dir)
	err = ss2.LoadIndex(indexPath)
	require.NoError(t, err)

	assert.NotNil(t, ss2.GetIndex())
	assert.NotEmpty(t, ss2.GetIndex().Functions)
}

// ---------------------------------------------------------------------------
// SemanticSearchTool.Execute
// ---------------------------------------------------------------------------

func TestSemanticSearchTool_Metadata(t *testing.T) {
	t.Parallel()

	tool := newTestSemanticSearchTool(t, t.TempDir())
	assert.Equal(t, "semantic_search", tool.Name())
	assert.NotEmpty(t, tool.Description())
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "semantic_search", def.Name)
}

func TestSemanticSearchTool_Execute_BuildAndSearch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "main", "main.go", `
// Calculator provides math operations
type Calculator struct{}

func (c *Calculator) Add(a, b int) int { return a + b }
`)

	tool := newTestSemanticSearchTool(t, dir)

	// Build index
	result, source, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "build_index",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)

	// Search
	result, source, err = tool.Execute(context.Background(), map[string]interface{}{
		"action": "search",
		"query":  "calculator add",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)
}

func TestSemanticSearchTool_Execute_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)

	tool := newTestSemanticSearchTool(t, dir)

	// Build first
	_, _, err := tool.Execute(context.Background(), map[string]interface{}{"action": "build_index"})
	require.NoError(t, err)

	indexPath := filepath.Join(dir, "test_index.json")

	// Save
	result, _, err := tool.Execute(context.Background(), map[string]interface{}{
		"action":     "save_index",
		"index_path": indexPath,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "index_saved")

	// Load
	result, _, err = tool.Execute(context.Background(), map[string]interface{}{
		"action":     "load_index",
		"index_path": indexPath,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "index_loaded")
}

func TestSemanticSearchTool_Execute_UnknownAction(t *testing.T) {
	t.Parallel()

	tool := newTestSemanticSearchTool(t, t.TempDir())
	_, _, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "unknown",
	})
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// containsString (package-level helper)
// ---------------------------------------------------------------------------

func TestContainsString(t *testing.T) {
	t.Parallel()

	assert.True(t, containsString([]string{"a", "b", "c"}, "b"))
	assert.False(t, containsString([]string{"a", "b", "c"}, "d"))
	assert.False(t, containsString(nil, "a"))
}
