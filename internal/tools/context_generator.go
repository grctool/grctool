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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
)

// ContextGenerator generates AI-friendly context about the codebase
type ContextGenerator struct {
	rootPath string
	logger   logger.Logger
}

// NewContextGenerator creates a new context generator
func NewContextGenerator(rootPath string) *ContextGenerator {
	return &ContextGenerator{
		rootPath: rootPath,
		logger:   logger.WithComponent("context-generator"),
	}
}

// CodebaseSummary contains high-level metrics and structure
type CodebaseSummary struct {
	GeneratedAt  time.Time             `json:"generated_at"`
	TotalFiles   int                   `json:"total_files"`
	GoFiles      int                   `json:"go_files"`
	TestFiles    int                   `json:"test_files"`
	Packages     []PackageInfo         `json:"packages"`
	Dependencies []string              `json:"dependencies"`
	KeyMetrics   map[string]int        `json:"key_metrics"`
	Architecture ArchitecturalOverview `json:"architecture"`
}

// PackageInfo describes a Go package
type PackageInfo struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Files       []string `json:"files"`
	Interfaces  []string `json:"interfaces"`
	Functions   int      `json:"functions"`
	LOC         int      `json:"loc"`
	Description string   `json:"description"`
}

// ArchitecturalOverview describes the system architecture
type ArchitecturalOverview struct {
	LayeredStructure map[string][]string `json:"layered_structure"`
	CoreModules      []string            `json:"core_modules"`
	EntryPoints      []string            `json:"entry_points"`
	TestStrategy     string              `json:"test_strategy"`
}

// InterfaceMapping contains interface definitions and implementations
type InterfaceMapping struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Interfaces  []InterfaceInfo `json:"interfaces"`
}

// InterfaceInfo describes a Go interface
type InterfaceInfo struct {
	Name         string   `json:"name"`
	Package      string   `json:"package"`
	File         string   `json:"file"`
	Methods      []string `json:"methods"`
	Implementers []string `json:"implementers"`
	Description  string   `json:"description"`
}

// DependencyGraph contains package and module dependencies
type DependencyGraph struct {
	GeneratedAt      time.Time                   `json:"generated_at"`
	Modules          []ModuleDependency          `json:"modules"`
	InternalPackages []InternalPackageDependency `json:"internal_packages"`
}

// ModuleDependency describes a Go module dependency
type ModuleDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Direct  bool   `json:"direct"`
}

// InternalPackageDependency describes internal package relationships
type InternalPackageDependency struct {
	Package    string   `json:"package"`
	ImportedBy []string `json:"imported_by"`
	Imports    []string `json:"imports"`
	Centrality int      `json:"centrality"`
}

// RecentChanges contains git history with semantic meaning
type RecentChanges struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Changes     []FileChange  `json:"changes"`
	Summary     ChangeSummary `json:"summary"`
}

// FileChange describes a recent file modification
type FileChange struct {
	File         string    `json:"file"`
	LastModified time.Time `json:"last_modified"`
	ChangeType   string    `json:"change_type"`
	Impact       string    `json:"impact"`
	Package      string    `json:"package"`
}

// ChangeSummary provides high-level change insights
type ChangeSummary struct {
	HotSpots         []string `json:"hot_spots"`
	RecentFeatures   []string `json:"recent_features"`
	ModifiedPackages []string `json:"modified_packages"`
}

// GenerateCodebaseSummary creates a comprehensive codebase overview
func (cg *ContextGenerator) GenerateCodebaseSummary(ctx context.Context) (*CodebaseSummary, error) {
	cg.logger.Info("generating codebase summary")
	start := time.Now()

	summary := &CodebaseSummary{
		GeneratedAt:  time.Now(),
		KeyMetrics:   make(map[string]int),
		Packages:     []PackageInfo{},
		Dependencies: []string{},
		Architecture: ArchitecturalOverview{
			LayeredStructure: make(map[string][]string),
		},
	}

	// Count files
	if err := cg.countFiles(summary); err != nil {
		return nil, fmt.Errorf("failed to count files: %w", err)
	}

	// Analyze packages
	if err := cg.analyzePackages(summary); err != nil {
		return nil, fmt.Errorf("failed to analyze packages: %w", err)
	}

	// Read dependencies
	if err := cg.readDependencies(summary); err != nil {
		cg.logger.Warn("failed to read dependencies", logger.Field{Key: "error", Value: err.Error()})
	}

	// Analyze architecture
	cg.analyzeArchitecture(summary)

	cg.logger.Info("codebase summary generated",
		logger.Field{Key: "duration_ms", Value: time.Since(start).Milliseconds()},
		logger.Field{Key: "packages", Value: len(summary.Packages)},
		logger.Field{Key: "go_files", Value: summary.GoFiles})

	return summary, nil
}

// GenerateInterfaceMapping creates a map of all interfaces and their implementations
func (cg *ContextGenerator) GenerateInterfaceMapping(ctx context.Context) (*InterfaceMapping, error) {
	cg.logger.Info("generating interface mapping")
	start := time.Now()

	mapping := &InterfaceMapping{
		GeneratedAt: time.Now(),
		Interfaces:  []InterfaceInfo{},
	}

	// Walk through Go files and extract interfaces
	err := filepath.Walk(cg.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		interfaces, err := cg.extractInterfacesFromFile(path)
		if err != nil {
			cg.logger.Warn("failed to extract interfaces", logger.Field{Key: "file", Value: path}, logger.Field{Key: "error", Value: err.Error()})
			return nil // Continue processing other files
		}

		mapping.Interfaces = append(mapping.Interfaces, interfaces...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	cg.logger.Info("interface mapping generated",
		logger.Field{Key: "duration_ms", Value: time.Since(start).Milliseconds()},
		logger.Field{Key: "interfaces", Value: len(mapping.Interfaces)})

	return mapping, nil
}

// GenerateDependencyGraph creates a dependency relationship graph
func (cg *ContextGenerator) GenerateDependencyGraph(ctx context.Context) (*DependencyGraph, error) {
	cg.logger.Info("generating dependency graph")
	start := time.Now()

	graph := &DependencyGraph{
		GeneratedAt:      time.Now(),
		Modules:          []ModuleDependency{},
		InternalPackages: []InternalPackageDependency{},
	}

	// Read go.mod for module dependencies
	if err := cg.readModuleDependencies(graph); err != nil {
		cg.logger.Warn("failed to read module dependencies", logger.Field{Key: "error", Value: err.Error()})
	}

	// Analyze internal package dependencies
	if err := cg.analyzeInternalDependencies(graph); err != nil {
		return nil, fmt.Errorf("failed to analyze internal dependencies: %w", err)
	}

	cg.logger.Info("dependency graph generated",
		logger.Field{Key: "duration_ms", Value: time.Since(start).Milliseconds()},
		logger.Field{Key: "modules", Value: len(graph.Modules)},
		logger.Field{Key: "internal_packages", Value: len(graph.InternalPackages)})

	return graph, nil
}

// countFiles counts different types of files in the codebase
func (cg *ContextGenerator) countFiles(summary *CodebaseSummary) error {
	err := filepath.Walk(cg.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.Contains(path, "vendor/") || strings.Contains(path, ".git/") {
			return nil
		}

		summary.TotalFiles++

		if strings.HasSuffix(path, ".go") {
			summary.GoFiles++
			if strings.HasSuffix(path, "_test.go") {
				summary.TestFiles++
			}
		}

		return nil
	})

	summary.KeyMetrics["total_files"] = summary.TotalFiles
	summary.KeyMetrics["go_files"] = summary.GoFiles
	summary.KeyMetrics["test_files"] = summary.TestFiles
	summary.KeyMetrics["test_coverage_ratio"] = (summary.TestFiles * 100) / maxInt(summary.GoFiles, 1)

	return err
}

// analyzePackages analyzes Go packages in the codebase
func (cg *ContextGenerator) analyzePackages(summary *CodebaseSummary) error {
	packageMap := make(map[string]*PackageInfo)

	err := filepath.Walk(cg.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		// Parse file to get package info
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Continue processing other files
		}

		pkgName := node.Name.Name
		relPath, _ := filepath.Rel(cg.rootPath, filepath.Dir(path))

		if _, exists := packageMap[pkgName]; !exists {
			packageMap[pkgName] = &PackageInfo{
				Name:       pkgName,
				Path:       relPath,
				Files:      []string{},
				Interfaces: []string{},
				Functions:  0,
			}
		}

		pkg := packageMap[pkgName]
		pkg.Files = append(pkg.Files, filepath.Base(path))

		// Count functions and interfaces
		ast.Inspect(node, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				pkg.Functions++
			case *ast.TypeSpec:
				if _, ok := x.Type.(*ast.InterfaceType); ok {
					pkg.Interfaces = append(pkg.Interfaces, x.Name.Name)
				}
			}
			return true
		})

		return nil
	})

	// Convert map to slice and sort
	for _, pkg := range packageMap {
		summary.Packages = append(summary.Packages, *pkg)
	}

	sort.Slice(summary.Packages, func(i, j int) bool {
		return summary.Packages[i].Name < summary.Packages[j].Name
	})

	return err
}

// extractInterfacesFromFile extracts interface definitions from a Go file
func (cg *ContextGenerator) extractInterfacesFromFile(filePath string) ([]InterfaceInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var interfaces []InterfaceInfo

	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if iface, ok := ts.Type.(*ast.InterfaceType); ok {
				info := InterfaceInfo{
					Name:    ts.Name.Name,
					Package: node.Name.Name,
					File:    filePath,
					Methods: []string{},
				}

				// Extract method signatures
				for _, method := range iface.Methods.List {
					if method.Names != nil {
						for _, name := range method.Names {
							info.Methods = append(info.Methods, name.Name)
						}
					}
				}

				interfaces = append(interfaces, info)
			}
		}
		return true
	})

	return interfaces, nil
}

// readDependencies reads module dependencies from go.mod
func (cg *ContextGenerator) readDependencies(summary *CodebaseSummary) error {
	goModPath := filepath.Join(cg.rootPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		if line == ")" {
			inRequireBlock = false
			continue
		}

		if inRequireBlock && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				summary.Dependencies = append(summary.Dependencies, parts[0])
			}
		}
	}

	return nil
}

// analyzeArchitecture identifies architectural patterns
func (cg *ContextGenerator) analyzeArchitecture(summary *CodebaseSummary) {
	// Identify layered structure
	summary.Architecture.LayeredStructure["cmd"] = []string{}
	summary.Architecture.LayeredStructure["internal"] = []string{}
	summary.Architecture.LayeredStructure["test"] = []string{}

	for _, pkg := range summary.Packages {
		if strings.HasPrefix(pkg.Path, "cmd/") {
			summary.Architecture.LayeredStructure["cmd"] = append(summary.Architecture.LayeredStructure["cmd"], pkg.Name)
		} else if strings.HasPrefix(pkg.Path, "internal/") {
			summary.Architecture.LayeredStructure["internal"] = append(summary.Architecture.LayeredStructure["internal"], pkg.Name)
		} else if strings.HasPrefix(pkg.Path, "test/") {
			summary.Architecture.LayeredStructure["test"] = append(summary.Architecture.LayeredStructure["test"], pkg.Name)
		}
	}

	// Identify core modules
	coreModules := []string{"models", "config", "logger", "storage", "tools", "tugboat"}
	for _, module := range coreModules {
		for _, pkg := range summary.Packages {
			if pkg.Name == module {
				summary.Architecture.CoreModules = append(summary.Architecture.CoreModules, module)
				break
			}
		}
	}

	// Set test strategy
	summary.Architecture.TestStrategy = "4-tier: unit/integration/functional/e2e with VCR framework"
}

// readModuleDependencies reads module dependencies from go.mod
func (cg *ContextGenerator) readModuleDependencies(graph *DependencyGraph) error {
	// This would typically run `go list -m all` but for simplicity, read go.mod
	goModPath := filepath.Join(cg.rootPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		if line == ")" {
			inRequireBlock = false
			continue
		}

		if inRequireBlock && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				graph.Modules = append(graph.Modules, ModuleDependency{
					Name:    parts[0],
					Version: parts[1],
					Direct:  true,
				})
			}
		}
	}

	return nil
}

// analyzeInternalDependencies analyzes internal package dependencies
func (cg *ContextGenerator) analyzeInternalDependencies(graph *DependencyGraph) error {
	packageDeps := make(map[string]InternalPackageDependency)

	err := filepath.Walk(cg.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		pkgName := node.Name.Name
		if _, exists := packageDeps[pkgName]; !exists {
			packageDeps[pkgName] = InternalPackageDependency{
				Package:    pkgName,
				ImportedBy: []string{},
				Imports:    []string{},
			}
		}

		dep := packageDeps[pkgName]

		// Analyze imports
		for _, imp := range node.Imports {
			impPath := strings.Trim(imp.Path.Value, "\"")
			if strings.Contains(impPath, "grctool/") {
				parts := strings.Split(impPath, "/")
				if len(parts) > 0 {
					importedPkg := parts[len(parts)-1]
					dep.Imports = append(dep.Imports, importedPkg)
				}
			}
		}

		packageDeps[pkgName] = dep
		return nil
	})

	// Calculate centrality (how many packages import this package)
	for pkgName, dep := range packageDeps {
		for _, otherDep := range packageDeps {
			for _, imp := range otherDep.Imports {
				if imp == pkgName {
					dep.ImportedBy = append(dep.ImportedBy, otherDep.Package)
				}
			}
		}
		dep.Centrality = len(dep.ImportedBy)
		graph.InternalPackages = append(graph.InternalPackages, dep)
	}

	// Sort by centrality (most central first)
	sort.Slice(graph.InternalPackages, func(i, j int) bool {
		return graph.InternalPackages[i].Centrality > graph.InternalPackages[j].Centrality
	})

	return err
}

// ContextGeneratorTool implements the Tool interface for CLI integration
type ContextGeneratorTool struct {
	generator *ContextGenerator
	logger    logger.Logger
}

// NewContextGeneratorTool creates a new context generator tool
func NewContextGeneratorTool(rootPath string) *ContextGeneratorTool {
	return &ContextGeneratorTool{
		generator: NewContextGenerator(rootPath),
		logger:    logger.WithComponent("context-tool"),
	}
}

// Name returns the tool name
func (t *ContextGeneratorTool) Name() string {
	return "context_generator"
}

// Description returns the tool description
func (t *ContextGeneratorTool) Description() string {
	return "Generate AI-friendly context about the codebase structure, dependencies, and architecture"
}

// Execute runs the context generation (legacy interface)
func (t *ContextGeneratorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	contextType, ok := params["context_type"].(string)
	if !ok {
		contextType = "summary"
	}

	outputPath, _ := params["output_path"].(string)

	var result interface{}
	var err error

	switch contextType {
	case "summary":
		result, err = t.generator.GenerateCodebaseSummary(ctx)
	case "interfaces":
		result, err = t.generator.GenerateInterfaceMapping(ctx)
	case "dependencies":
		result, err = t.generator.GenerateDependencyGraph(ctx)
	default:
		return "", nil, fmt.Errorf("unknown context_type: %s", contextType)
	}

	if err != nil {
		return "", nil, err
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Save to file if output_path specified
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			return "", nil, fmt.Errorf("failed to write output file: %w", err)
		}
	}

	source := &models.EvidenceSource{
		Type:        "context_generation",
		Resource:    fmt.Sprintf("context_generator_%s", contextType),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"context_type": contextType,
			"output_path":  outputPath,
		},
	}

	return string(jsonData), source, nil
}

// GetClaudeToolDefinition returns the Claude tool definition
func (t *ContextGeneratorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "context_generator",
		Description: "Generate AI-friendly context about the codebase",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"context_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of context to generate",
					"enum":        []string{"summary", "interfaces", "dependencies"},
				},
				"output_path": map[string]interface{}{
					"type":        "string",
					"description": "Optional path to save the context output",
				},
			},
			"required": []string{},
		},
	}
}

// ExecuteTyped implements the TypedTool interface
func (t *ContextGeneratorTool) ExecuteTyped(ctx context.Context, req types.Request) (types.Response, error) {
	// For now, delegate to the legacy Execute method
	adapter := types.NewToolAdapter(t)
	return adapter.ExecuteTyped(ctx, req)
}

// Helper function for maxInt (Go 1.21+)
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
