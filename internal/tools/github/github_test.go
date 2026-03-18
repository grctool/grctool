package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/grctool/grctool/internal/tools/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers: build a GitHubClient pointing at an httptest server
// ---------------------------------------------------------------------------

func newTestClient(t *testing.T, handler http.Handler) *GitHubClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	log := testhelpers.NewStubLogger()
	return &GitHubClient{
		config: &config.GitHubToolConfig{
			Repository: "test-org/test-repo",
			APIToken:   "fake-token",
			MaxIssues:  50,
		},
		httpClient: srv.Client(),
		logger:     log,
		baseURL:    srv.URL,
		graphqlURL: srv.URL + "/graphql",
		cacheDir:   t.TempDir(),
	}
}

// ---------------------------------------------------------------------------
// NewGitHubClientWithAuth
// ---------------------------------------------------------------------------

func TestNewGitHubClientWithAuth_UsesInjectedProvider(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	cfg := &config.Config{
		Storage: config.StorageConfig{DataDir: t.TempDir()},
		Auth: config.AuthConfig{
			GitHub:   config.GitHubAuthConfig{Token: "should-not-be-used"},
			CacheDir: t.TempDir(),
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{Repository: "org/repo"},
			},
		},
	}

	// Inject a custom auth provider
	injected := auth.NewGitHubAuthProvider("injected-token", t.TempDir(), log)
	client := NewGitHubClientWithAuth(cfg, log, injected)

	require.NotNil(t, client)
	assert.Equal(t, "github", client.authProvider.Name())
	// The injected provider should be the one used, not one built from config
	assert.Same(t, injected, client.authProvider)
	client.Close()
}

func TestNewGitHubClientWithAuth_NilFallsBackToConfig(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	cfg := &config.Config{
		Storage: config.StorageConfig{DataDir: t.TempDir()},
		Auth: config.AuthConfig{
			GitHub:   config.GitHubAuthConfig{Token: "from-config"},
			CacheDir: t.TempDir(),
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{Repository: "org/repo"},
			},
		},
	}

	client := NewGitHubClientWithAuth(cfg, log, nil)

	require.NotNil(t, client)
	assert.Equal(t, "github", client.authProvider.Name())
	// Auth provider should have been created from config
	status := client.authProvider.GetStatus(context.Background())
	assert.True(t, status.TokenPresent)
	client.Close()
}

// ---------------------------------------------------------------------------
// GitHubSearchResults.TotalCount
// ---------------------------------------------------------------------------

func TestGitHubSearchResultsTotalCount(t *testing.T) {
	t.Parallel()

	r := &GitHubSearchResults{
		Commits:      make([]GitHubCommitResult, 2),
		Workflows:    make([]GitHubWorkflowResult, 3),
		Issues:       make([]models.GitHubIssueResult, 1),
		PullRequests: make([]models.GitHubIssueResult, 4),
	}
	assert.Equal(t, 10, r.TotalCount())

	empty := &GitHubSearchResults{}
	assert.Equal(t, 0, empty.TotalCount())
}

// ---------------------------------------------------------------------------
// REST API: GetRepositoryCollaborators
// ---------------------------------------------------------------------------

func TestGetRepositoryCollaborators(t *testing.T) {
	t.Parallel()

	collaboratorsJSON := `[
		{"login":"alice","id":1,"permissions":{"admin":true,"push":true,"pull":true}},
		{"login":"bob","id":2,"permissions":{"admin":false,"push":true,"pull":true}}
	]`
	permissionJSON := `{"permission":"admin","user":{"login":"alice"}}`

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/collaborators", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(collaboratorsJSON))
	})
	mux.HandleFunc("/repos/org/repo/collaborators/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/permission") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(permissionJSON))
		}
	})

	client := newTestClient(t, mux)
	collabs, err := client.GetRepositoryCollaborators(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.Len(t, collabs, 2)
	assert.Equal(t, "alice", collabs[0].Login)
}

// ---------------------------------------------------------------------------
// REST API: GetRepositoryTeams
// ---------------------------------------------------------------------------

func TestGetRepositoryTeams(t *testing.T) {
	t.Parallel()

	teamsJSON := `[{"id":1,"name":"devs","slug":"devs","permission":"push"}]`
	membersJSON := `[{"login":"charlie","id":3,"role":"member"}]`

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/teams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(teamsJSON))
	})
	mux.HandleFunc("/teams/1/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(membersJSON))
	})

	client := newTestClient(t, mux)
	teams, err := client.GetRepositoryTeams(context.Background(), "org", "repo")
	require.NoError(t, err)
	require.Len(t, teams, 1)
	assert.Equal(t, "devs", teams[0].Name)
	assert.Len(t, teams[0].Members, 1)
}

// ---------------------------------------------------------------------------
// REST API: GetRepositoryBranches
// ---------------------------------------------------------------------------

func TestGetRepositoryBranches(t *testing.T) {
	t.Parallel()

	branchesJSON := `[{"name":"main","protected":true},{"name":"dev","protected":false}]`

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/branches", func(w http.ResponseWriter, r *http.Request) {
		// Only match the exact path, not sub-paths
		if r.URL.Path == "/repos/org/repo/branches" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(branchesJSON))
		}
	})
	mux.HandleFunc("/repos/org/repo/branches/main/protection", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"enabled":true}`))
	})
	mux.HandleFunc("/repos/org/repo/branches/dev/protection", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := newTestClient(t, mux)
	branches, err := client.GetRepositoryBranches(context.Background(), "org", "repo")
	require.NoError(t, err)
	require.Len(t, branches, 2)
	assert.Equal(t, "main", branches[0].Name)
	assert.True(t, branches[0].Protected)
}

// ---------------------------------------------------------------------------
// REST API: GetBranchProtection (404 = no protection)
// ---------------------------------------------------------------------------

func TestGetBranchProtection_NotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/branches/dev/protection", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := newTestClient(t, mux)
	protection, err := client.GetBranchProtection(context.Background(), "org", "repo", "dev")
	require.NoError(t, err)
	assert.Nil(t, protection)
}

// ---------------------------------------------------------------------------
// REST API: GetDeploymentEnvironments
// ---------------------------------------------------------------------------

func TestGetDeploymentEnvironments(t *testing.T) {
	t.Parallel()

	envJSON := `{
		"total_count": 1,
		"environments": [
			{
				"id": 42,
				"name": "production",
				"protection_rules": [
					{
						"id": 1,
						"type": "required_reviewers",
						"reviewers": [{"type": "User", "reviewer": {"login": "admin"}}]
					}
				],
				"created_at": "2025-01-01T00:00:00Z",
				"updated_at": "2025-01-02T00:00:00Z"
			}
		]
	}`

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/environments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(envJSON))
	})

	client := newTestClient(t, mux)
	envs, err := client.GetDeploymentEnvironments(context.Background(), "org", "repo")
	require.NoError(t, err)
	require.Len(t, envs, 1)
	assert.Equal(t, "production", envs[0].Name)
	assert.Equal(t, "42", envs[0].ID)
	require.Len(t, envs[0].ProtectionRules, 1)
	assert.Equal(t, "required_reviewers", envs[0].ProtectionRules[0].Type)
}

func TestGetDeploymentEnvironments_404(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/environments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := newTestClient(t, mux)
	envs, err := client.GetDeploymentEnvironments(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.Empty(t, envs)
}

// ---------------------------------------------------------------------------
// REST API: GetRepositorySecurity
// ---------------------------------------------------------------------------

func TestGetRepositorySecurity(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/vulnerability-alerts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // enabled
	})
	mux.HandleFunc("/repos/org/repo/automated-security-fixes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"enabled": true}`))
	})
	mux.HandleFunc("/repos/org/repo", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/org/repo" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"security_and_analysis":{"secret_scanning":{"status":"enabled"}}}`))
		}
	})
	mux.HandleFunc("/repos/org/repo/code-scanning/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	client := newTestClient(t, mux)
	settings, err := client.GetRepositorySecurity(context.Background(), "org", "repo")
	require.NoError(t, err)
	assert.True(t, settings.VulnerabilityAlertsEnabled)
	assert.True(t, settings.AutomatedSecurityFixesEnabled)
	assert.True(t, settings.SecretScanningEnabled)
	assert.True(t, settings.CodeScanningEnabled)
}

// ---------------------------------------------------------------------------
// REST API: GetOrganizationMembers
// ---------------------------------------------------------------------------

func TestGetOrganizationMembers(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/myorg/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"login":"member1","id":10}]`))
	})

	client := newTestClient(t, mux)
	members, err := client.GetOrganizationMembers(context.Background(), "myorg")
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "member1", members[0].Login)
}

// ---------------------------------------------------------------------------
// REST API: error handling
// ---------------------------------------------------------------------------

func TestMakeRESTRequest_HTTPError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/collaborators", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"server error"}`))
	})

	client := newTestClient(t, mux)
	_, err := client.GetRepositoryCollaborators(context.Background(), "org", "repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestMakeRESTRequest_RateLimited(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/teams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		w.WriteHeader(http.StatusForbidden)
	})

	client := newTestClient(t, mux)
	_, err := client.GetRepositoryTeams(context.Background(), "org", "repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
}

// ---------------------------------------------------------------------------
// Relevance calculations
// ---------------------------------------------------------------------------

func TestCalculateCommitRelevance(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())

	commit := GitHubCommitResult{
		Commit: CommitData{
			Message: "fix security vulnerability in auth module",
			Author:  AuthorData{Date: time.Now().Add(-24 * time.Hour)}, // recent
		},
	}

	score := client.calculateCommitRelevance(commit, "security")
	assert.Greater(t, score, 0.5) // should get message match + recency + security keyword
}

func TestCalculateWorkflowRelevance(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())

	workflow := GitHubWorkflowResult{
		Name: "security-scan",
		Path: ".github/workflows/security-scan.yml",
	}

	score := client.calculateWorkflowRelevance(workflow, "security")
	assert.Greater(t, score, 0.5)
}

func TestCalculateIssueRelevance(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())

	issue := models.GitHubIssueResult{
		Title:     "Security audit findings",
		Body:      "Found several issues",
		State:     "open",
		Labels:    []models.GitHubLabel{{Name: "security"}},
		UpdatedAt: time.Now(),
	}

	score := client.calculateIssueRelevance(issue, "security", []string{"security"})
	assert.Greater(t, score, 0.5)
}

func TestCalculateIssueRelevance_NoLabels(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())

	issue := models.GitHubIssueResult{Title: "unrelated", Body: "nothing here", State: "closed"}
	score := client.calculateIssueRelevance(issue, "security", nil)
	assert.Equal(t, 0.0, score)
}

// ---------------------------------------------------------------------------
// Cache operations
// ---------------------------------------------------------------------------

func TestCacheRoundTrip(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())

	results := &GitHubSearchResults{
		Issues: []models.GitHubIssueResult{
			{Title: "cached issue", Number: 1},
		},
	}

	key := client.generateCacheKey("test", "issue", "", 10, nil)
	client.SaveToCache(key, results)

	loaded := client.LoadFromCache(key)
	require.NotNil(t, loaded)
	require.Len(t, loaded.Issues, 1)
	assert.Equal(t, "cached issue", loaded.Issues[0].Title)
}

func TestLoadFromCache_Missing(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())
	assert.Nil(t, client.LoadFromCache("nonexistent"))
}

// ---------------------------------------------------------------------------
// GenerateCorrelationID
// ---------------------------------------------------------------------------

func TestGenerateCorrelationID(t *testing.T) {
	t.Parallel()

	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	assert.True(t, strings.HasPrefix(id1, "github-"))
	assert.NotEqual(t, id1, id2) // should be unique
}

// ---------------------------------------------------------------------------
// GraphQL error handling
// ---------------------------------------------------------------------------

func TestMakeGraphQLRequest_HTTPError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Bad credentials"}`))
	})

	client := newTestClient(t, mux)
	_, err := client.makeGraphQLRequest(context.Background(), "{ viewer { login } }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestMakeGraphQLRequest_GraphQLErrors(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data":   nil,
			"errors": []map[string]interface{}{{"message": "field not found"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := newTestClient(t, mux)
	_, err := client.makeGraphQLRequest(context.Background(), "{ bad }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL errors")
}

// ---------------------------------------------------------------------------
// GitHubClient Close
// ---------------------------------------------------------------------------

func TestClientClose(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())
	client.rateLimiter = time.NewTicker(time.Second)
	client.Close() // should not panic
}

func TestClientClose_NilRateLimiter(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())
	client.rateLimiter = nil
	client.Close() // should not panic
}

// ---------------------------------------------------------------------------
// Tool metadata: Name, Description, GetClaudeToolDefinition
// ---------------------------------------------------------------------------

func TestGitHubPermissionsToolMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubPermissionsTool{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-permissions", tool.Name())
	assert.Contains(t, tool.Description(), "access controls")
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-permissions", def.Name)
	assert.NotNil(t, def.InputSchema)
}

func TestGitHubDeploymentAccessToolMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubDeploymentAccessTool{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-deployment-access", tool.Name())
	assert.Contains(t, tool.Description(), "deployment")
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-deployment-access", def.Name)
}

func TestGitHubSecurityFeaturesToolMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubSecurityFeaturesTool{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-security-features", tool.Name())
	assert.Contains(t, tool.Description(), "security feature")
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-security-features", def.Name)
}

func TestGitHubWorkflowAnalyzerMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubWorkflowAnalyzer{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-workflow-analyzer", tool.Name())
	assert.Contains(t, tool.Description(), "workflow")
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-workflow-analyzer", def.Name)
}

func TestGitHubReviewAnalyzerMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubReviewAnalyzer{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-review-analyzer", tool.Name())
	assert.Contains(t, tool.Description(), "review")
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-review-analyzer", def.Name)
}

func TestGitHubEnhancedToolMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubEnhancedTool{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-enhanced", tool.Name())
	assert.Contains(t, tool.Description(), "Enhanced")
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-enhanced", def.Name)
}

func TestGitHubToolMetadata(t *testing.T) {
	t.Parallel()
	tool := &GitHubTool{logger: testhelpers.NewStubLogger()}
	assert.Equal(t, "github-searcher", tool.Name())
	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "github-searcher", def.Name)
}

// ---------------------------------------------------------------------------
// Placeholder methods (permissions.go report generators)
// ---------------------------------------------------------------------------

func TestPermissionsToolPlaceholders(t *testing.T) {
	t.Parallel()

	tool := &GitHubPermissionsTool{logger: testhelpers.NewStubLogger()}
	matrix := &models.GitHubAccessControlMatrix{}

	assert.NotEmpty(t, tool.generateDetailedReport(matrix))
	assert.NotEmpty(t, tool.generatePermissionMatrix(matrix))
	assert.NotEmpty(t, tool.generateAccessSummary(matrix))
	assert.InDelta(t, 0.8, tool.calculateRelevance(matrix), 0.01)
}

func TestDeploymentAccessToolPlaceholders(t *testing.T) {
	t.Parallel()

	tool := &GitHubDeploymentAccessTool{logger: testhelpers.NewStubLogger()}
	info := &DeploymentAccessInfo{}

	assert.NotEmpty(t, tool.generateDeploymentMatrix(info))
	assert.NotEmpty(t, tool.generateDeploymentSummary(info))
	assert.NotEmpty(t, tool.generateDetailedDeploymentReport(info))
	assert.InDelta(t, 0.8, tool.calculateDeploymentRelevance(info), 0.01)
}

func TestDeploymentAccessTool_CountProtectedEnvironments(t *testing.T) {
	t.Parallel()

	tool := &GitHubDeploymentAccessTool{logger: testhelpers.NewStubLogger()}

	envs := []models.GitHubEnvironment{
		{Name: "prod", ProtectionRules: []models.GitHubEnvironmentProtection{{Type: "required_reviewers"}}},
		{Name: "staging"},
		{Name: "dev", ProtectionRules: []models.GitHubEnvironmentProtection{{Type: "wait_timer"}}},
	}
	assert.Equal(t, 2, tool.countProtectedEnvironments(envs))
	assert.Equal(t, 0, tool.countProtectedEnvironments(nil))
}

func TestSecurityFeaturesToolPlaceholders(t *testing.T) {
	t.Parallel()

	tool := &GitHubSecurityFeaturesTool{logger: testhelpers.NewStubLogger()}
	info := &SecurityFeaturesInfo{AllFeatures: make(map[string]SecurityFeatureDetail)}

	assert.NotEmpty(t, tool.generateSecurityMatrix(info))
	assert.NotEmpty(t, tool.generateSecuritySummary(info))
	assert.NotEmpty(t, tool.generateDetailedSecurityReport(info))
	assert.InDelta(t, 0.8, tool.calculateSecurityRelevance(info), 0.01)
	assert.InDelta(t, 0.8, tool.calculateSecurityScore(info), 0.01)
	assert.Empty(t, tool.generateSecurityRecommendations(info))
}

// ---------------------------------------------------------------------------
// GitHubTool report generation and relevance
// ---------------------------------------------------------------------------

func TestGitHubToolGenerateReport_Empty(t *testing.T) {
	t.Parallel()

	tool := &GitHubTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	assert.Equal(t, "No relevant GitHub issues found.", tool.generateReport(nil))
}

func TestGitHubToolGenerateReport_WithIssues(t *testing.T) {
	t.Parallel()

	tool := &GitHubTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	now := time.Now()
	issues := []models.GitHubIssueResult{
		{
			Number:    42,
			Title:     "Security fix",
			State:     "open",
			Body:      "Need to fix auth",
			Labels:    []models.GitHubLabel{{Name: "security"}},
			CreatedAt: now,
			UpdatedAt: now,
			URL:       "https://github.com/org/repo/issues/42",
			Relevance: 0.9,
		},
	}
	report := tool.generateReport(issues)
	assert.Contains(t, report, "Security fix")
	assert.Contains(t, report, "#42")
	assert.Contains(t, report, "security")
}

func TestGitHubToolCalculateOverallRelevance(t *testing.T) {
	t.Parallel()

	tool := &GitHubTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}

	assert.Equal(t, 0.0, tool.calculateOverallRelevance(nil))

	oneIssue := []models.GitHubIssueResult{{Relevance: 0.5}}
	assert.InDelta(t, 0.5, tool.calculateOverallRelevance(oneIssue), 0.01)

	// 5+ issues get a bonus
	fiveIssues := make([]models.GitHubIssueResult, 5)
	for i := range fiveIssues {
		fiveIssues[i].Relevance = 0.5
	}
	assert.InDelta(t, 0.7, tool.calculateOverallRelevance(fiveIssues), 0.01)
}

// ---------------------------------------------------------------------------
// GitHubEnhancedTool report and relevance
// ---------------------------------------------------------------------------

func TestGitHubEnhancedToolGenerateReport(t *testing.T) {
	t.Parallel()

	tool := &GitHubEnhancedTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}

	results := &GitHubSearchResults{
		Issues: []models.GitHubIssueResult{{Title: "test"}},
	}
	report := tool.generateEnhancedReport(results, "security", "all")
	assert.Contains(t, report, "Enhanced GitHub Security Evidence")
	assert.Contains(t, report, "Total Results: 1")
}

func TestGitHubEnhancedToolCalculateRelevance(t *testing.T) {
	t.Parallel()

	tool := &GitHubEnhancedTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}

	empty := &GitHubSearchResults{}
	assert.Equal(t, 0.0, tool.calculateEnhancedRelevance(empty, "test"))

	withResults := &GitHubSearchResults{Issues: []models.GitHubIssueResult{{Title: "a"}}}
	assert.InDelta(t, 0.8, tool.calculateEnhancedRelevance(withResults, "test"), 0.01)
}

// ---------------------------------------------------------------------------
// PerformEnhancedSearch - unsupported type (does not require network)
// ---------------------------------------------------------------------------

func TestPerformEnhancedSearch_UnsupportedType(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, http.NewServeMux())
	_, err := client.PerformEnhancedSearch(context.Background(), "test", "badtype", "", 10, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported search type")
}

// ---------------------------------------------------------------------------
// Workflow and Review analyzer placeholder methods
// ---------------------------------------------------------------------------

func TestWorkflowAnalyzerPlaceholders(t *testing.T) {
	t.Parallel()

	tool := &GitHubWorkflowAnalyzer{logger: testhelpers.NewStubLogger()}
	analysis := &models.GitHubWorkflowAnalysis{}

	report := tool.generateWorkflowReport(analysis, "full")
	assert.NotEmpty(t, report)
	assert.InDelta(t, 0.8, tool.calculateWorkflowRelevance(analysis), 0.01)
}

func TestReviewAnalyzerPlaceholders(t *testing.T) {
	t.Parallel()

	tool := &GitHubReviewAnalyzer{logger: testhelpers.NewStubLogger()}
	analysis := &models.GitHubPullRequestAnalysis{}

	report := tool.generateReviewReport(analysis, true)
	assert.NotEmpty(t, report)
	assert.InDelta(t, 0.8, tool.calculateReviewRelevance(analysis), 0.01)
}

// ---------------------------------------------------------------------------
// PermissionsTool helper: generateAccessSummaryData
// ---------------------------------------------------------------------------

func TestGenerateAccessSummaryData(t *testing.T) {
	t.Parallel()

	tool := &GitHubPermissionsTool{logger: testhelpers.NewStubLogger()}
	matrix := &models.GitHubAccessControlMatrix{}

	summary := tool.generateAccessSummaryData(matrix)
	// Placeholder returns empty summary
	assert.Equal(t, 0, summary.TotalCollaborators)
}

// ---------------------------------------------------------------------------
// DeploymentAccessTool: buildDeploymentAccessMatrix
// ---------------------------------------------------------------------------

func TestBuildDeploymentAccessMatrix(t *testing.T) {
	t.Parallel()

	tool := &GitHubDeploymentAccessTool{logger: testhelpers.NewStubLogger()}
	result := tool.buildDeploymentAccessMatrix(nil)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// SecurityFeaturesTool: buildComplianceMapping, analyzeSecurityPolicies, buildFeatureAnalysis
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Adapter pattern tests (using direct tool construction)
// ---------------------------------------------------------------------------

func TestGitHubAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubAdapter{tool: inner}

	assert.Equal(t, "github-searcher", adapter.Name())
	assert.Contains(t, adapter.Description(), "Search GitHub")
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-searcher", def.Name)
}

func TestGitHubEnhancedAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubEnhancedTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubEnhancedAdapter{tool: inner}

	assert.Equal(t, "github-enhanced", adapter.Name())
	assert.Contains(t, adapter.Description(), "Enhanced")
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-enhanced", def.Name)
}

func TestGitHubPermissionsAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubPermissionsTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubPermissionsAdapter{tool: inner}

	assert.Equal(t, "github-permissions", adapter.Name())
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-permissions", def.Name)
}

func TestGitHubDeploymentAccessAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubDeploymentAccessTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubDeploymentAccessAdapter{tool: inner}

	assert.Equal(t, "github-deployment-access", adapter.Name())
	assert.Contains(t, adapter.Description(), "deployment")
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-deployment-access", def.Name)
}

func TestGitHubSecurityFeaturesAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubSecurityFeaturesTool{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubSecurityFeaturesAdapter{tool: inner}

	assert.Equal(t, "github-security-features", adapter.Name())
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-security-features", def.Name)
}

func TestGitHubWorkflowAnalyzerAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubWorkflowAnalyzer{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubWorkflowAnalyzerAdapter{tool: inner}

	assert.Equal(t, "github-workflow-analyzer", adapter.Name())
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-workflow-analyzer", def.Name)
}

func TestGitHubReviewAnalyzerAdapterDelegation(t *testing.T) {
	t.Parallel()

	inner := &GitHubReviewAnalyzer{
		client: newTestClient(t, http.NewServeMux()),
		logger: testhelpers.NewStubLogger(),
	}
	adapter := &GitHubReviewAnalyzerAdapter{tool: inner}

	assert.Equal(t, "github-review-analyzer", adapter.Name())
	def := adapter.GetClaudeToolDefinition()
	assert.Equal(t, "github-review-analyzer", def.Name)
}

// ---------------------------------------------------------------------------
// LegacyToolMappings
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Constructors using minimal config (exercises NewGitHubClient path)
// ---------------------------------------------------------------------------

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Storage: config.StorageConfig{DataDir: t.TempDir()},
		Auth: config.AuthConfig{
			CacheDir: t.TempDir(),
			GitHub:   config.GitHubAuthConfig{Token: "test-token"},
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Repository: "test/repo",
					APIToken:   "test-token",
					MaxIssues:  50,
					RateLimit:  600, // fast for tests
				},
			},
		},
	}
}

func TestNewGitHubToolConstructors(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	cfg := testConfig(t)

	tools := []struct {
		name    string
		factory func(*config.Config, logger.Logger) types.LegacyTool
	}{
		{"github-searcher", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubTool(c, l) }},
		{"github-enhanced", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubEnhancedTool(c, l) }},
		{"github-permissions", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubPermissionsTool(c, l) }},
		{"github-deployment-access", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubDeploymentAccessTool(c, l) }},
		{"github-security-features", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubSecurityFeaturesTool(c, l) }},
		{"github-workflow-analyzer", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubWorkflowAnalyzer(c, l) }},
		{"github-review-analyzer", func(c *config.Config, l logger.Logger) types.LegacyTool { return NewGitHubReviewAnalyzer(c, l) }},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tool := tt.factory(cfg, log)
			require.NotNil(t, tool)
			assert.Equal(t, tt.name, tool.Name())
			assert.NotEmpty(t, tool.Description())
			def := tool.GetClaudeToolDefinition()
			assert.Equal(t, tt.name, def.Name)
		})
	}
}

func TestNewGitHubAdapterConstructors(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	cfg := testConfig(t)

	adapters := []struct {
		name    string
		factory func(*config.Config, logger.Logger) types.LegacyTool
	}{
		{"github-searcher", NewGitHubAdapter},
		{"github-enhanced", NewGitHubEnhancedAdapter},
		{"github-permissions", NewGitHubPermissionsAdapter},
		{"github-deployment-access", NewGitHubDeploymentAccessAdapter},
		{"github-security-features", NewGitHubSecurityFeaturesAdapter},
		{"github-workflow-analyzer", NewGitHubWorkflowAnalyzerAdapter},
		{"github-review-analyzer", NewGitHubReviewAnalyzerAdapter},
	}

	for _, tt := range adapters {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := tt.factory(cfg, log)
			require.NotNil(t, adapter)
			assert.Equal(t, tt.name, adapter.Name())
			assert.NotEmpty(t, adapter.Description())
		})
	}
}

func TestGetGitHubTool(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	log := testhelpers.NewStubLogger()

	tool := GetGitHubTool("github-permissions", cfg, log)
	require.NotNil(t, tool)
	assert.Equal(t, "github-permissions", tool.Name())

	missing := GetGitHubTool("nonexistent", cfg, log)
	assert.Nil(t, missing)
}

func TestGetAllGitHubTools(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	log := testhelpers.NewStubLogger()

	allTools := GetAllGitHubTools(cfg, log)
	assert.Len(t, allTools, len(LegacyToolMappings))
	for name := range LegacyToolMappings {
		_, exists := allTools[name]
		assert.True(t, exists, "missing tool: %s", name)
	}
}

func TestLegacyToolMappings(t *testing.T) {
	t.Parallel()

	expected := []string{
		"github", "github-enhanced", "github_searcher",
		"github-permissions", "github-deployment-access", "github-security-features",
		"github-workflow-analyzer", "github-review-analyzer",
	}
	for _, name := range expected {
		_, exists := LegacyToolMappings[name]
		assert.True(t, exists, "expected mapping for %s", name)
	}
}

// ---------------------------------------------------------------------------
// Adapter Execute delegation (cover the one-liner Execute methods)
// ---------------------------------------------------------------------------

type fakeGitHubTool struct {
	name string
}

func (f *fakeGitHubTool) Name() string                          { return f.name }
func (f *fakeGitHubTool) Description() string                   { return "fake" }
func (f *fakeGitHubTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{Name: f.name}
}
func (f *fakeGitHubTool) Execute(_ context.Context, _ map[string]interface{}) (string, *models.EvidenceSource, error) {
	return "result", &models.EvidenceSource{Type: f.name}, nil
}

func TestAdapterExecuteDelegation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		adapter types.LegacyTool
	}{
		{"GitHubAdapter", &GitHubAdapter{tool: &fakeGitHubTool{name: "a"}}},
		{"GitHubEnhancedAdapter", &GitHubEnhancedAdapter{tool: &fakeGitHubTool{name: "b"}}},
		{"GitHubPermissionsAdapter", &GitHubPermissionsAdapter{tool: &fakeGitHubTool{name: "c"}}},
		{"GitHubDeploymentAccessAdapter", &GitHubDeploymentAccessAdapter{tool: &fakeGitHubTool{name: "d"}}},
		{"GitHubSecurityFeaturesAdapter", &GitHubSecurityFeaturesAdapter{tool: &fakeGitHubTool{name: "e"}}},
		{"GitHubWorkflowAnalyzerAdapter", &GitHubWorkflowAnalyzerAdapter{tool: &fakeGitHubTool{name: "f"}}},
		{"GitHubReviewAnalyzerAdapter", &GitHubReviewAnalyzerAdapter{tool: &fakeGitHubTool{name: "g"}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, src, err := tc.adapter.Execute(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, "result", result)
			assert.NotNil(t, src)
		})
	}
}

func TestSecurityFeaturesToolHelpers(t *testing.T) {
	t.Parallel()

	tool := &GitHubSecurityFeaturesTool{logger: testhelpers.NewStubLogger()}
	info := &SecurityFeaturesInfo{AllFeatures: make(map[string]SecurityFeatureDetail)}

	mapping := tool.buildComplianceMapping(info)
	assert.NotNil(t, mapping)

	policies, err := tool.analyzeSecurityPolicies(context.Background(), "org", "repo")
	assert.NoError(t, err)
	assert.Empty(t, policies)

	// buildFeatureAnalysis should not panic
	tool.buildFeatureAnalysis(info)
}
