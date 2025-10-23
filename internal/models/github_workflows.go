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

package models

import (
	"time"
)

// GitHubWorkflowAnalysis represents the comprehensive analysis of GitHub Actions workflows
type GitHubWorkflowAnalysis struct {
	Repository      string                   `json:"repository"`
	AnalysisDate    time.Time                `json:"analysis_date"`
	WorkflowFiles   []GitHubWorkflowFile     `json:"workflow_files"`
	SecurityScans   []GitHubSecurityScan     `json:"security_scans"`
	ApprovalRules   []GitHubApprovalRule     `json:"approval_rules"`
	Statistics      GitHubWorkflowStatistics `json:"statistics"`
	ComplianceScore float64                  `json:"compliance_score"`
	RecommendedBy   string                   `json:"recommended_by"`
}

// GitHubWorkflowFile represents a parsed GitHub Actions workflow file
type GitHubWorkflowFile struct {
	Name            string                       `json:"name"`
	Path            string                       `json:"path"`
	Content         string                       `json:"content,omitempty"`
	Triggers        []GitHubWorkflowTrigger      `json:"triggers"`
	Jobs            []GitHubWorkflowJob          `json:"jobs"`
	SecuritySteps   []GitHubWorkflowSecurityStep `json:"security_steps"`
	ApprovalSteps   []GitHubWorkflowApprovalStep `json:"approval_steps"`
	Environment     string                       `json:"environment,omitempty"`
	RequiredReviews int                          `json:"required_reviews"`
	LastModified    time.Time                    `json:"last_modified"`
	HTMLURL         string                       `json:"html_url"`
	Relevance       float64                      `json:"relevance"`
	ComplianceRules []GitHubComplianceRule       `json:"compliance_rules"`
}

// GitHubWorkflowTrigger represents workflow execution triggers
type GitHubWorkflowTrigger struct {
	Event    string                 `json:"event"` // push, pull_request, schedule, etc.
	Branches []string               `json:"branches,omitempty"`
	Paths    []string               `json:"paths,omitempty"`
	Schedule string                 `json:"schedule,omitempty"`
	Types    []string               `json:"types,omitempty"`
	Filters  map[string]interface{} `json:"filters,omitempty"`
}

// GitHubWorkflowJob represents a job within a workflow
type GitHubWorkflowJob struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	RunsOn          string                     `json:"runs_on"`
	Environment     string                     `json:"environment,omitempty"`
	Steps           []GitHubWorkflowStep       `json:"steps"`
	Needs           []string                   `json:"needs,omitempty"`
	If              string                     `json:"if,omitempty"`
	Permissions     map[string]string          `json:"permissions,omitempty"`
	Outputs         map[string]string          `json:"outputs,omitempty"`
	Strategy        *GitHubWorkflowJobStrategy `json:"strategy,omitempty"`
	TimeoutMinutes  int                        `json:"timeout_minutes,omitempty"`
	ContinueOnError bool                       `json:"continue_on_error"`
}

// GitHubWorkflowStep represents a step within a job
type GitHubWorkflowStep struct {
	ID               string                 `json:"id,omitempty"`
	Name             string                 `json:"name,omitempty"`
	Uses             string                 `json:"uses,omitempty"`
	Run              string                 `json:"run,omitempty"`
	With             map[string]interface{} `json:"with,omitempty"`
	Env              map[string]string      `json:"env,omitempty"`
	If               string                 `json:"if,omitempty"`
	ContinueOnError  bool                   `json:"continue_on_error"`
	TimeoutMinutes   int                    `json:"timeout_minutes,omitempty"`
	WorkingDirectory string                 `json:"working_directory,omitempty"`
	Shell            string                 `json:"shell,omitempty"`
}

// GitHubWorkflowJobStrategy represents job execution strategy
type GitHubWorkflowJobStrategy struct {
	Matrix      map[string]interface{} `json:"matrix,omitempty"`
	FailFast    *bool                  `json:"fail_fast,omitempty"`
	MaxParallel int                    `json:"max_parallel,omitempty"`
}

// GitHubWorkflowSecurityStep represents security-related workflow steps
type GitHubWorkflowSecurityStep struct {
	StepName        string                 `json:"step_name"`
	Action          string                 `json:"action"`
	Purpose         string                 `json:"purpose"` // code_scanning, dependency_check, secret_scanning, etc.
	Tool            string                 `json:"tool"`    // CodeQL, Dependabot, Trivy, etc.
	Configuration   map[string]interface{} `json:"configuration"`
	FailureHandling string                 `json:"failure_handling"` // fail, warn, ignore
	ReportLocation  string                 `json:"report_location,omitempty"`
	Frequency       string                 `json:"frequency"` // on_push, on_pr, scheduled
}

// GitHubWorkflowApprovalStep represents approval and review requirements
type GitHubWorkflowApprovalStep struct {
	StepName          string   `json:"step_name"`
	Environment       string   `json:"environment"`
	RequiredReviewers []string `json:"required_reviewers"`
	RequiredTeams     []string `json:"required_teams"`
	RequiredChecks    []string `json:"required_checks"`
	WaitTimer         int      `json:"wait_timer,omitempty"` // in minutes
	PreventSelfReview bool     `json:"prevent_self_review"`
}

// GitHubSecurityScan represents security scanning configuration
type GitHubSecurityScan struct {
	Type            string                 `json:"type"` // codeql, dependabot, secret_scanning
	Enabled         bool                   `json:"enabled"`
	Configuration   map[string]interface{} `json:"configuration"`
	Languages       []string               `json:"languages,omitempty"`
	Schedule        string                 `json:"schedule,omitempty"`
	ExcludePatterns []string               `json:"exclude_patterns,omitempty"`
	FailureAction   string                 `json:"failure_action"` // fail, warn, ignore
	LastRun         *time.Time             `json:"last_run,omitempty"`
	Status          string                 `json:"status"` // active, disabled, error
}

// GitHubApprovalRule represents branch protection and approval rules
type GitHubApprovalRule struct {
	Branch                       string   `json:"branch"`
	RequiredReviews              int      `json:"required_reviews"`
	RequireCodeOwnerReviews      bool     `json:"require_code_owner_reviews"`
	DismissStaleReviews          bool     `json:"dismiss_stale_reviews"`
	RestrictPushes               bool     `json:"restrict_pushes"`
	RequiredStatusChecks         []string `json:"required_status_checks"`
	RequireUpToDateBranches      bool     `json:"require_up_to_date_branches"`
	RequireSignedCommits         bool     `json:"require_signed_commits"`
	RequireLinearHistory         bool     `json:"require_linear_history"`
	AllowForcePushes             bool     `json:"allow_force_pushes"`
	AllowDeletions               bool     `json:"allow_deletions"`
	RestrictedPushUsers          []string `json:"restricted_push_users,omitempty"`
	RestrictedPushTeams          []string `json:"restricted_push_teams,omitempty"`
	RequiredApprovingReviewCount int      `json:"required_approving_review_count"`
	BypassUsers                  []string `json:"bypass_users,omitempty"`
	BypassTeams                  []string `json:"bypass_teams,omitempty"`
}

// GitHubWorkflowStatistics represents workflow analysis statistics
type GitHubWorkflowStatistics struct {
	TotalWorkflows      int                                  `json:"total_workflows"`
	SecurityWorkflows   int                                  `json:"security_workflows"`
	DeploymentWorkflows int                                  `json:"deployment_workflows"`
	TestWorkflows       int                                  `json:"test_workflows"`
	SecurityStepsCount  int                                  `json:"security_steps_count"`
	ApprovalStepsCount  int                                  `json:"approval_steps_count"`
	AverageJobsPerFlow  float64                              `json:"average_jobs_per_flow"`
	ToolsUsed           map[string]int                       `json:"tools_used"` // tool name -> usage count
	TriggerTypes        map[string]int                       `json:"trigger_types"`
	ComplianceBreakdown map[string]GitHubComplianceBreakdown `json:"compliance_breakdown"`
}

// GitHubComplianceRule represents specific compliance rules for workflows
type GitHubComplianceRule struct {
	RuleID      string `json:"rule_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`     // pass, fail, warning, not_applicable
	Evidence    string `json:"evidence"`   // What was found that satisfies or violates the rule
	Framework   string `json:"framework"`  // SOC2, ISO27001, etc.
	ControlID   string `json:"control_id"` // Specific control this rule maps to
}

// GitHubComplianceBreakdown represents compliance status by category
type GitHubComplianceBreakdown struct {
	Category     string   `json:"category"` // code_review, security_scanning, deployment_controls
	Passed       int      `json:"passed"`
	Failed       int      `json:"failed"`
	Warnings     int      `json:"warnings"`
	NotTested    int      `json:"not_tested"`
	Score        float64  `json:"score"`        // 0.0-1.0
	Requirements []string `json:"requirements"` // List of requirements for this category
}

// GitHubPullRequestAnalysis represents comprehensive PR review analysis
type GitHubPullRequestAnalysis struct {
	Repository       string                    `json:"repository"`
	AnalysisDate     time.Time                 `json:"analysis_date"`
	DateRange        GitHubDateRange           `json:"date_range"`
	PullRequests     []GitHubPullRequestDetail `json:"pull_requests"`
	ReviewMetrics    GitHubReviewMetrics       `json:"review_metrics"`
	ComplianceScore  float64                   `json:"compliance_score"`
	Recommendations  []GitHubRecommendation    `json:"recommendations"`
	ApprovalPatterns GitHubApprovalPatterns    `json:"approval_patterns"`
}

// GitHubDateRange represents the analysis date range
type GitHubDateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Period    string    `json:"period"` // "30d", "90d", "1y", etc.
}

// GitHubPullRequestDetail represents detailed PR information
type GitHubPullRequestDetail struct {
	Number             int                      `json:"number"`
	Title              string                   `json:"title"`
	State              string                   `json:"state"`
	Author             GitHubUser               `json:"author"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
	MergedAt           *time.Time               `json:"merged_at,omitempty"`
	ClosedAt           *time.Time               `json:"closed_at,omitempty"`
	Labels             []string                 `json:"labels"`
	Reviews            []GitHubPRReview         `json:"reviews"`
	RequestedReviewers []GitHubUser             `json:"requested_reviewers"`
	RequestedTeams     []GitHubTeam             `json:"requested_teams"`
	ChangedFiles       int                      `json:"changed_files"`
	Additions          int                      `json:"additions"`
	Deletions          int                      `json:"deletions"`
	StatusChecks       []GitHubStatusCheck      `json:"status_checks"`
	Branch             GitHubBranchInfo         `json:"branch"`
	BaseBranch         GitHubBranchInfo         `json:"base_branch"`
	ApprovalTimeline   []GitHubApprovalEvent    `json:"approval_timeline"`
	ComplianceStatus   GitHubPRComplianceStatus `json:"compliance_status"`
	HTMLURL            string                   `json:"html_url"`
	SecurityRelevant   bool                     `json:"security_relevant"`
	Relevance          float64                  `json:"relevance"`
}

// Note: GitHubUser and GitHubTeam are defined in github_permissions.go

// GitHubPRReview represents a pull request review
type GitHubPRReview struct {
	ID                int        `json:"id"`
	User              GitHubUser `json:"user"`
	State             string     `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	Body              string     `json:"body,omitempty"`
	SubmittedAt       time.Time  `json:"submitted_at"`
	CommitID          string     `json:"commit_id"`
	AuthorAssociation string     `json:"author_association"` // OWNER, MEMBER, COLLABORATOR, etc.
}

// GitHubStatusCheck represents a status check or CI result
type GitHubStatusCheck struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`     // pending, success, failure, error
	Conclusion  string     `json:"conclusion"` // success, failure, cancelled, skipped, etc.
	URL         string     `json:"url,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// GitHubBranchInfo represents branch information
type GitHubBranchInfo struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
	Ref  string `json:"ref"`
}

// GitHubApprovalEvent represents an event in the approval timeline
type GitHubApprovalEvent struct {
	Type        string     `json:"type"` // review_requested, reviewed, approved, changes_requested
	Actor       GitHubUser `json:"actor"`
	Timestamp   time.Time  `json:"timestamp"`
	Action      string     `json:"action"`                 // requested, submitted, dismissed
	ReviewState string     `json:"review_state,omitempty"` // APPROVED, CHANGES_REQUESTED, etc.
}

// GitHubPRComplianceStatus represents PR compliance with policies
type GitHubPRComplianceStatus struct {
	RequiredReviewsMet    bool     `json:"required_reviews_met"`
	CodeOwnerApprovalMet  bool     `json:"code_owner_approval_met"`
	StatusChecksPassed    bool     `json:"status_checks_passed"`
	BranchUpToDate        bool     `json:"branch_up_to_date"`
	ConversationsResolved bool     `json:"conversations_resolved"`
	ComplianceScore       float64  `json:"compliance_score"`
	ViolatedRules         []string `json:"violated_rules"`
	SatisfiedRules        []string `json:"satisfied_rules"`
}

// GitHubReviewMetrics represents overall review metrics
type GitHubReviewMetrics struct {
	TotalPRs              int                            `json:"total_prs"`
	MergedPRs             int                            `json:"merged_prs"`
	OpenPRs               int                            `json:"open_prs"`
	ClosedPRs             int                            `json:"closed_prs"`
	AverageReviewTime     time.Duration                  `json:"average_review_time"`
	AverageApprovalTime   time.Duration                  `json:"average_approval_time"`
	AverageReviewsPerPR   float64                        `json:"average_reviews_per_pr"`
	ReviewParticipation   map[string]int                 `json:"review_participation"` // user -> review count
	ApprovalRate          float64                        `json:"approval_rate"`
	ChangeRequestRate     float64                        `json:"change_request_rate"`
	FirstTimeApprovalRate float64                        `json:"first_time_approval_rate"`
	SecurityPRs           int                            `json:"security_prs"`
	ComplianceRate        float64                        `json:"compliance_rate"`
	ReviewerStats         map[string]GitHubReviewerStats `json:"reviewer_stats"`
	TimeBasedMetrics      GitHubTimeBasedMetrics         `json:"time_based_metrics"`
}

// GitHubReviewerStats represents statistics for individual reviewers
type GitHubReviewerStats struct {
	ReviewsGiven        int           `json:"reviews_given"`
	ApprovalsGiven      int           `json:"approvals_given"`
	ChangeRequestsGiven int           `json:"change_requests_given"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ReviewQuality       float64       `json:"review_quality"` // Based on follow-up changes needed
}

// GitHubTimeBasedMetrics represents time-based analysis
type GitHubTimeBasedMetrics struct {
	PRsByMonth         map[string]int           `json:"prs_by_month"`
	ReviewsByMonth     map[string]int           `json:"reviews_by_month"`
	ApprovalsByMonth   map[string]int           `json:"approvals_by_month"`
	AverageTimeToMerge map[string]time.Duration `json:"average_time_to_merge"` // by month
}

// GitHubApprovalPatterns represents approval workflow patterns
type GitHubApprovalPatterns struct {
	CommonApprovers       []GitHubUser          `json:"common_approvers"`
	ApprovalChains        []GitHubApprovalChain `json:"approval_chains"`
	BotUsage              map[string]int        `json:"bot_usage"`
	AutoMergeUsage        int                   `json:"auto_merge_usage"`
	RequiredCheckPatterns map[string]int        `json:"required_check_patterns"`
}

// GitHubApprovalChain represents common approval sequences
type GitHubApprovalChain struct {
	Pattern    []string `json:"pattern"`    // sequence of approver types/roles
	Frequency  int      `json:"frequency"`  // how often this pattern occurs
	Percentage float64  `json:"percentage"` // percentage of total PRs
}

// GitHubRecommendation represents process improvement recommendations
type GitHubRecommendation struct {
	Type        string   `json:"type"`     // security, efficiency, compliance
	Priority    string   `json:"priority"` // high, medium, low
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Evidence    string   `json:"evidence"` // What data supports this recommendation
	ActionItems []string `json:"action_items"`
	Impact      string   `json:"impact"` // Expected impact of implementing
}
