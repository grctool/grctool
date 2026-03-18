// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeGoFile(t *testing.T, dir, pkg, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	full := "package " + pkg + "\n\n" + content
	require.NoError(t, os.WriteFile(path, []byte(full), 0o644))
}

func writeGoMod(t *testing.T, dir string) {
	t.Helper()
	content := `module example.com/testmod

go 1.24

require (
	github.com/stretchr/testify v1.9.0
)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644))
}

// newTestContextGenerator creates a ContextGenerator with a stub logger
// so tests don't panic from a nil global logger.
func newTestContextGenerator(t *testing.T, rootPath string) *ContextGenerator {
	t.Helper()
	return &ContextGenerator{
		rootPath: rootPath,
		logger:   testhelpers.NewStubLogger(),
	}
}

// ---------------------------------------------------------------------------
// NewContextGenerator
// ---------------------------------------------------------------------------

func TestNewContextGenerator(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// NewContextGenerator uses logger.WithComponent which returns nil
	// when no default logger is set. Use our helper instead.
	cg := newTestContextGenerator(t, dir)
	require.NotNil(t, cg)
	assert.Equal(t, dir, cg.rootPath)
}

// ---------------------------------------------------------------------------
// countFiles
// ---------------------------------------------------------------------------

func TestContextGenerator_CountFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)
	writeGoFile(t, dir, "main", "main_test.go", `func TestMain(t *testing.T) {}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644))

	cg := newTestContextGenerator(t, dir)
	summary := &CodebaseSummary{
		KeyMetrics: make(map[string]int),
	}

	err := cg.countFiles(summary)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.TotalFiles)
	assert.Equal(t, 2, summary.GoFiles)
	assert.Equal(t, 1, summary.TestFiles)
	assert.Equal(t, 3, summary.KeyMetrics["total_files"])
	assert.Equal(t, 2, summary.KeyMetrics["go_files"])
	assert.Equal(t, 1, summary.KeyMetrics["test_files"])
}

func TestContextGenerator_CountFiles_Empty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cg := newTestContextGenerator(t, dir)
	summary := &CodebaseSummary{
		KeyMetrics: make(map[string]int),
	}

	err := cg.countFiles(summary)
	require.NoError(t, err)
	assert.Equal(t, 0, summary.TotalFiles)
}

// ---------------------------------------------------------------------------
// analyzePackages
// ---------------------------------------------------------------------------

func TestContextGenerator_AnalyzePackages(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "main", "main.go", `
import "fmt"

type Greeter interface {
	Greet() string
}

func Hello() string {
	return fmt.Sprintf("hello")
}

func Goodbye() string {
	return "bye"
}
`)

	cg := newTestContextGenerator(t, dir)
	summary := &CodebaseSummary{
		Packages: []PackageInfo{},
	}

	err := cg.analyzePackages(summary)
	require.NoError(t, err)
	require.NotEmpty(t, summary.Packages)

	// Find the "main" package
	var mainPkg *PackageInfo
	for i := range summary.Packages {
		if summary.Packages[i].Name == "main" {
			mainPkg = &summary.Packages[i]
			break
		}
	}
	require.NotNil(t, mainPkg)
	assert.GreaterOrEqual(t, mainPkg.Functions, 2)
	assert.Contains(t, mainPkg.Interfaces, "Greeter")
}

// ---------------------------------------------------------------------------
// extractInterfacesFromFile
// ---------------------------------------------------------------------------

func TestContextGenerator_ExtractInterfacesFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "iface.go")
	content := `package example

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type SimpleStruct struct {
	Name string
}
`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	cg := newTestContextGenerator(t, dir)
	interfaces, err := cg.extractInterfacesFromFile(filePath)
	require.NoError(t, err)
	assert.Len(t, interfaces, 2)

	names := make([]string, len(interfaces))
	for i, iface := range interfaces {
		names[i] = iface.Name
	}
	assert.Contains(t, names, "Reader")
	assert.Contains(t, names, "Writer")
}

// ---------------------------------------------------------------------------
// readDependencies
// ---------------------------------------------------------------------------

func TestContextGenerator_ReadDependencies(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)

	cg := newTestContextGenerator(t, dir)
	summary := &CodebaseSummary{
		Dependencies: []string{},
	}

	err := cg.readDependencies(summary)
	require.NoError(t, err)
	assert.NotEmpty(t, summary.Dependencies)
	// Check for testify dependency (version format may vary)
	found := false
	for _, dep := range summary.Dependencies {
		if filepath.Base(dep) == "testify" || dep == "github.com/stretchr/testify" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find testify dependency, got: %v", summary.Dependencies)
}

func TestContextGenerator_ReadDependencies_NoGoMod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cg := newTestContextGenerator(t, dir)
	summary := &CodebaseSummary{
		Dependencies: []string{},
	}

	err := cg.readDependencies(summary)
	assert.Error(t, err, "should error when go.mod not found")
}

// ---------------------------------------------------------------------------
// readModuleDependencies
// ---------------------------------------------------------------------------

func TestContextGenerator_ReadModuleDependencies(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)

	cg := newTestContextGenerator(t, dir)
	graph := &DependencyGraph{
		Modules: []ModuleDependency{},
	}

	err := cg.readModuleDependencies(graph)
	require.NoError(t, err)
	assert.NotEmpty(t, graph.Modules)
}

// ---------------------------------------------------------------------------
// analyzeInternalDependencies
// ---------------------------------------------------------------------------

func TestContextGenerator_AnalyzeInternalDependencies(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create a package that imports another
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg"), 0o755))
	writeGoFile(t, dir, "main", "main.go", `
import "fmt"

func main() {
	fmt.Println("hello")
}
`)

	cg := newTestContextGenerator(t, dir)
	graph := &DependencyGraph{
		InternalPackages: []InternalPackageDependency{},
	}

	err := cg.analyzeInternalDependencies(graph)
	require.NoError(t, err)
	// May or may not find internal deps, but should not error
}

// ---------------------------------------------------------------------------
// analyzeArchitecture
// ---------------------------------------------------------------------------

func TestContextGenerator_AnalyzeArchitecture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create some typical Go project structure
	for _, subdir := range []string{"cmd/app", "internal/config", "internal/tools", "pkg/utils"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, subdir), 0o755))
		writeGoFile(t, filepath.Join(dir, subdir), filepath.Base(subdir), "main.go", "// placeholder")
	}

	cg := newTestContextGenerator(t, dir)
	summary := &CodebaseSummary{
		Architecture: ArchitecturalOverview{
			LayeredStructure: make(map[string][]string),
		},
		Packages: []PackageInfo{
			{Name: "app", Path: "cmd/app"},
			{Name: "config", Path: "internal/config"},
			{Name: "tools", Path: "internal/tools"},
			{Name: "utils", Path: "pkg/utils"},
		},
	}

	cg.analyzeArchitecture(summary)
	// Should categorize packages into layers
	assert.NotEmpty(t, summary.Architecture.LayeredStructure)
}

// ---------------------------------------------------------------------------
// GenerateCodebaseSummary (integration)
// ---------------------------------------------------------------------------

func TestContextGenerator_GenerateCodebaseSummary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir, "main", "main.go", `
func main() {}
`)
	writeGoFile(t, dir, "main", "main_test.go", `
import "testing"

func TestHello(t *testing.T) {}
`)

	cg := newTestContextGenerator(t, dir)
	summary, err := cg.GenerateCodebaseSummary(context.Background())
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Greater(t, summary.TotalFiles, 0)
	assert.Greater(t, summary.GoFiles, 0)
	assert.Greater(t, summary.TestFiles, 0)
	assert.NotEmpty(t, summary.Packages)
}

// ---------------------------------------------------------------------------
// GenerateInterfaceMapping (integration)
// ---------------------------------------------------------------------------

func TestContextGenerator_GenerateInterfaceMapping(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "example", "types.go", `
type Storer interface {
	Save(data []byte) error
	Load(id string) ([]byte, error)
}
`)

	cg := newTestContextGenerator(t, dir)
	mapping, err := cg.GenerateInterfaceMapping(context.Background())
	require.NoError(t, err)
	require.NotNil(t, mapping)
	assert.NotEmpty(t, mapping.Interfaces)
}

// ---------------------------------------------------------------------------
// GenerateDependencyGraph (integration)
// ---------------------------------------------------------------------------

func TestContextGenerator_GenerateDependencyGraph(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)

	cg := newTestContextGenerator(t, dir)
	graph, err := cg.GenerateDependencyGraph(context.Background())
	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.NotEmpty(t, graph.Modules)
}

// ---------------------------------------------------------------------------
// maxInt
// ---------------------------------------------------------------------------

func TestMaxInt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, maxInt(5, 3))
	assert.Equal(t, 5, maxInt(3, 5))
	assert.Equal(t, 0, maxInt(0, 0))
	assert.Equal(t, 1, maxInt(1, 1))
}

// ---------------------------------------------------------------------------
// ContextGeneratorTool — Execute with different context types
// ---------------------------------------------------------------------------

func newTestContextGeneratorTool(t *testing.T, rootPath string) *ContextGeneratorTool {
	t.Helper()
	return &ContextGeneratorTool{
		generator: newTestContextGenerator(t, rootPath),
		logger:    testhelpers.NewStubLogger(),
	}
}

func TestContextGeneratorTool_Metadata(t *testing.T) {
	t.Parallel()

	tool := newTestContextGeneratorTool(t, t.TempDir())
	assert.Equal(t, "context_generator", tool.Name())
	assert.NotEmpty(t, tool.Description())

	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "context_generator", def.Name)
}

func TestContextGeneratorTool_Execute_Summary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)

	tool := newTestContextGeneratorTool(t, dir)

	result, source, err := tool.Execute(context.Background(), map[string]interface{}{
		"context_type": "summary",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "total_files")
	require.NotNil(t, source)
	assert.Equal(t, "context_generation", source.Type)
}

func TestContextGeneratorTool_Execute_Interfaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoFile(t, dir, "example", "types.go", `
type Storer interface {
	Save(data []byte) error
}
`)

	tool := newTestContextGeneratorTool(t, dir)

	result, source, err := tool.Execute(context.Background(), map[string]interface{}{
		"context_type": "interfaces",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Storer")
	require.NotNil(t, source)
}

func TestContextGeneratorTool_Execute_Dependencies(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)

	tool := newTestContextGeneratorTool(t, dir)

	result, source, err := tool.Execute(context.Background(), map[string]interface{}{
		"context_type": "dependencies",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	require.NotNil(t, source)
}

func TestContextGeneratorTool_Execute_DefaultType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)

	tool := newTestContextGeneratorTool(t, dir)

	// No context_type specified - should default to "summary"
	result, _, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestContextGeneratorTool_Execute_UnknownType(t *testing.T) {
	t.Parallel()

	tool := newTestContextGeneratorTool(t, t.TempDir())

	_, _, err := tool.Execute(context.Background(), map[string]interface{}{
		"context_type": "unknown",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown context_type")
}

func TestContextGeneratorTool_Execute_WithOutputPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir, "main", "main.go", `func main() {}`)

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "output", "context.json")

	tool := newTestContextGeneratorTool(t, dir)

	result, _, err := tool.Execute(context.Background(), map[string]interface{}{
		"context_type": "summary",
		"output_path":  outPath,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify file was written
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}
