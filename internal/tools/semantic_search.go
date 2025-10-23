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
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
)

// SemanticSearch provides semantic search capabilities for the codebase
type SemanticSearch struct {
	rootPath string
	index    *CodeIndex
	logger   logger.Logger
}

// CodeIndex contains indexed code elements for search
type CodeIndex struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Files       []SearchFileInfo       `json:"files"`
	Functions   []FunctionInfo         `json:"functions"`
	Types       []TypeInfo             `json:"types"`
	Interfaces  []InterfaceInfo        `json:"interfaces"`
	Comments    []CommentInfo          `json:"comments"`
	Keywords    map[string][]Reference `json:"keywords"`
}

// SearchFileInfo describes a file in the codebase
type SearchFileInfo struct {
	Path         string    `json:"path"`
	Package      string    `json:"package"`
	Imports      []string  `json:"imports"`
	LOC          int       `json:"loc"`
	Purpose      string    `json:"purpose"`
	Keywords     []string  `json:"keywords"`
	LastModified time.Time `json:"last_modified"`
}

// FunctionInfo describes a function or method
type FunctionInfo struct {
	Name        string   `json:"name"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Receiver    string   `json:"receiver,omitempty"`
	Parameters  []string `json:"parameters"`
	ReturnTypes []string `json:"return_types"`
	IsExported  bool     `json:"is_exported"`
	DocComment  string   `json:"doc_comment"`
	Purpose     string   `json:"purpose"`
	Keywords    []string `json:"keywords"`
}

// TypeInfo describes a type definition
type TypeInfo struct {
	Name       string   `json:"name"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Kind       string   `json:"kind"` // struct, interface, alias, etc.
	Fields     []string `json:"fields,omitempty"`
	Methods    []string `json:"methods,omitempty"`
	IsExported bool     `json:"is_exported"`
	DocComment string   `json:"doc_comment"`
	Purpose    string   `json:"purpose"`
	Keywords   []string `json:"keywords"`
}

// CommentInfo describes a significant comment
type CommentInfo struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Text     string   `json:"text"`
	Type     string   `json:"type"` // doc, todo, fixme, note, etc.
	Context  string   `json:"context"`
	Keywords []string `json:"keywords"`
}

// Reference points to a location in the code
type Reference struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Context string `json:"context"`
	Type    string `json:"type"` // function, type, comment, etc.
}

// SearchResult represents a search match
type SearchResult struct {
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	Context   string   `json:"context"`
	Relevance float64  `json:"relevance"`
	Keywords  []string `json:"keywords"`
}

// SearchQuery defines a search request
type SearchQuery struct {
	Query        string   `json:"query"`
	Types        []string `json:"types,omitempty"`         // function, type, interface, comment
	Files        []string `json:"files,omitempty"`         // filter by file patterns
	Packages     []string `json:"packages,omitempty"`      // filter by packages
	ExportedOnly bool     `json:"exported_only,omitempty"` // only exported symbols
	Limit        int      `json:"limit,omitempty"`         // max results
}

// NewSemanticSearch creates a new semantic search engine
func NewSemanticSearch(rootPath string) *SemanticSearch {
	return &SemanticSearch{
		rootPath: rootPath,
		logger:   logger.WithComponent("semantic-search"),
	}
}

// BuildIndex creates a searchable index of the codebase
func (s *SemanticSearch) BuildIndex(ctx context.Context) error {
	s.logger.Info("building semantic search index")
	start := time.Now()

	index := &CodeIndex{
		GeneratedAt: time.Now(),
		Files:       []SearchFileInfo{},
		Functions:   []FunctionInfo{},
		Types:       []TypeInfo{},
		Interfaces:  []InterfaceInfo{},
		Comments:    []CommentInfo{},
		Keywords:    make(map[string][]Reference),
	}

	err := filepath.Walk(s.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		if err := s.indexFile(path, index); err != nil {
			s.logger.Warn("failed to index file", logger.Field{Key: "file", Value: path}, logger.Field{Key: "error", Value: err.Error()})
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Build keyword index
	s.buildKeywordIndex(index)

	s.index = index

	s.logger.Info("semantic search index built",
		logger.Field{Key: "duration_ms", Value: time.Since(start).Milliseconds()},
		logger.Field{Key: "files", Value: len(index.Files)},
		logger.Field{Key: "functions", Value: len(index.Functions)},
		logger.Field{Key: "types", Value: len(index.Types)},
		logger.Field{Key: "keywords", Value: len(index.Keywords)})

	return nil
}

// Search performs semantic search on the indexed codebase
func (s *SemanticSearch) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	if s.index == nil {
		return nil, fmt.Errorf("index not built - call BuildIndex first")
	}

	s.logger.Debug("performing semantic search", logger.Field{Key: "query", Value: query.Query})

	var results []SearchResult
	queryTerms := s.tokenize(strings.ToLower(query.Query))

	// Search functions
	if len(query.Types) == 0 || containsString(query.Types, "function") {
		results = append(results, s.searchFunctions(queryTerms, query)...)
	}

	// Search types
	if len(query.Types) == 0 || containsString(query.Types, "type") {
		results = append(results, s.searchTypes(queryTerms, query)...)
	}

	// Search interfaces
	if len(query.Types) == 0 || containsString(query.Types, "interface") {
		results = append(results, s.searchInterfaces(queryTerms, query)...)
	}

	// Search comments
	if len(query.Types) == 0 || containsString(query.Types, "comment") {
		results = append(results, s.searchComments(queryTerms, query)...)
	}

	// Sort by relevance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	s.logger.Debug("search completed",
		logger.Field{Key: "results", Value: len(results)},
		logger.Field{Key: "query", Value: query.Query})

	return results, nil
}

// GetIndex returns the current code index (for inspection/debugging)
func (s *SemanticSearch) GetIndex() *CodeIndex {
	return s.index
}

// SaveIndex saves the index to a file
func (s *SemanticSearch) SaveIndex(path string) error {
	if s.index == nil {
		return fmt.Errorf("no index to save")
	}

	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	s.logger.Info("index saved", logger.Field{Key: "path", Value: path})
	return nil
}

// LoadIndex loads an index from a file
func (s *SemanticSearch) LoadIndex(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	var index CodeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	s.index = &index
	s.logger.Info("index loaded", logger.Field{Key: "path", Value: path})
	return nil
}

// indexFile processes a single Go file and adds it to the index
func (s *SemanticSearch) indexFile(filePath string, index *CodeIndex) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Create file info
	fileInfo := SearchFileInfo{
		Path:         filePath,
		Package:      node.Name.Name,
		Imports:      []string{},
		LastModified: info.ModTime(),
	}

	// Extract imports
	for _, imp := range node.Imports {
		impPath := strings.Trim(imp.Path.Value, "\"")
		fileInfo.Imports = append(fileInfo.Imports, impPath)
	}

	// Count lines
	content, err := os.ReadFile(filePath)
	if err == nil {
		fileInfo.LOC = strings.Count(string(content), "\n") + 1
	}

	// Infer purpose from file name and package
	fileInfo.Purpose = s.inferFilePurpose(filePath, node)
	fileInfo.Keywords = s.extractFileKeywords(node)

	index.Files = append(index.Files, fileInfo)

	// Process declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			s.indexFunction(x, filePath, fset, index)
		case *ast.GenDecl:
			s.indexGenDecl(x, filePath, fset, index)
		}
		return true
	})

	// Process comments
	s.indexComments(node.Comments, filePath, fset, index)

	return nil
}

// indexFunction processes a function declaration
func (s *SemanticSearch) indexFunction(fn *ast.FuncDecl, filePath string, fset *token.FileSet, index *CodeIndex) {
	pos := fset.Position(fn.Pos())

	funcInfo := FunctionInfo{
		Name:        fn.Name.Name,
		File:        filePath,
		Line:        pos.Line,
		IsExported:  ast.IsExported(fn.Name.Name),
		Parameters:  []string{},
		ReturnTypes: []string{},
	}

	// Extract receiver (for methods)
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		if len(fn.Recv.List[0].Names) > 0 {
			funcInfo.Receiver = fn.Recv.List[0].Names[0].Name
		}
	}

	// Extract parameters
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			paramType := s.typeToString(param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					funcInfo.Parameters = append(funcInfo.Parameters, name.Name+" "+paramType)
				}
			} else {
				funcInfo.Parameters = append(funcInfo.Parameters, paramType)
			}
		}
	}

	// Extract return types
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			funcInfo.ReturnTypes = append(funcInfo.ReturnTypes, s.typeToString(result.Type))
		}
	}

	// Extract doc comment
	if fn.Doc != nil {
		funcInfo.DocComment = fn.Doc.Text()
	}

	// Infer purpose and keywords
	funcInfo.Purpose = s.inferFunctionPurpose(funcInfo)
	funcInfo.Keywords = s.extractFunctionKeywords(funcInfo)

	index.Functions = append(index.Functions, funcInfo)
}

// indexGenDecl processes a general declaration (types, consts, vars)
func (s *SemanticSearch) indexGenDecl(decl *ast.GenDecl, filePath string, fset *token.FileSet, index *CodeIndex) {
	for _, spec := range decl.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			s.indexTypeSpec(spec, decl, filePath, fset, index)
		}
	}
}

// indexTypeSpec processes a type specification
func (s *SemanticSearch) indexTypeSpec(spec *ast.TypeSpec, decl *ast.GenDecl, filePath string, fset *token.FileSet, index *CodeIndex) {
	pos := fset.Position(spec.Pos())

	typeInfo := TypeInfo{
		Name:       spec.Name.Name,
		File:       filePath,
		Line:       pos.Line,
		IsExported: ast.IsExported(spec.Name.Name),
		Fields:     []string{},
		Methods:    []string{},
	}

	// Determine type kind and extract fields
	switch t := spec.Type.(type) {
	case *ast.StructType:
		typeInfo.Kind = "struct"
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldType := s.typeToString(field.Type)
				if len(field.Names) > 0 {
					for _, name := range field.Names {
						typeInfo.Fields = append(typeInfo.Fields, name.Name+" "+fieldType)
					}
				} else {
					// Anonymous field
					typeInfo.Fields = append(typeInfo.Fields, fieldType)
				}
			}
		}
	case *ast.InterfaceType:
		typeInfo.Kind = "interface"
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				if len(method.Names) > 0 {
					for _, name := range method.Names {
						typeInfo.Methods = append(typeInfo.Methods, name.Name)
					}
				}
			}
		}

		// Also add to interface index
		interfaceInfo := InterfaceInfo{
			Name:    spec.Name.Name,
			Package: filepath.Base(filepath.Dir(filePath)),
			File:    filePath,
			Methods: typeInfo.Methods,
		}
		index.Interfaces = append(index.Interfaces, interfaceInfo)
	default:
		typeInfo.Kind = "alias"
	}

	// Extract doc comment
	if decl.Doc != nil {
		typeInfo.DocComment = decl.Doc.Text()
	}

	// Infer purpose and keywords
	typeInfo.Purpose = s.inferTypePurpose(typeInfo)
	typeInfo.Keywords = s.extractTypeKeywords(typeInfo)

	index.Types = append(index.Types, typeInfo)
}

// indexComments processes comment groups
func (s *SemanticSearch) indexComments(comments []*ast.CommentGroup, filePath string, fset *token.FileSet, index *CodeIndex) {
	for _, cg := range comments {
		if cg == nil {
			continue
		}

		pos := fset.Position(cg.Pos())
		text := cg.Text()

		// Skip short or trivial comments
		if len(strings.TrimSpace(text)) < 10 {
			continue
		}

		commentInfo := CommentInfo{
			File: filePath,
			Line: pos.Line,
			Text: text,
			Type: s.classifyComment(text),
		}

		commentInfo.Keywords = s.extractCommentKeywords(text)
		index.Comments = append(index.Comments, commentInfo)
	}
}

// buildKeywordIndex creates a reverse index from keywords to references
func (s *SemanticSearch) buildKeywordIndex(index *CodeIndex) {
	// Index function keywords
	for _, fn := range index.Functions {
		for _, keyword := range fn.Keywords {
			index.Keywords[keyword] = append(index.Keywords[keyword], Reference{
				File:    fn.File,
				Line:    fn.Line,
				Context: fn.Name,
				Type:    "function",
			})
		}
	}

	// Index type keywords
	for _, typ := range index.Types {
		for _, keyword := range typ.Keywords {
			index.Keywords[keyword] = append(index.Keywords[keyword], Reference{
				File:    typ.File,
				Line:    typ.Line,
				Context: typ.Name,
				Type:    "type",
			})
		}
	}

	// Index comment keywords
	for _, comment := range index.Comments {
		for _, keyword := range comment.Keywords {
			index.Keywords[keyword] = append(index.Keywords[keyword], Reference{
				File:    comment.File,
				Line:    comment.Line,
				Context: comment.Type,
				Type:    "comment",
			})
		}
	}
}

// Search implementation methods

func (s *SemanticSearch) searchFunctions(queryTerms []string, query SearchQuery) []SearchResult {
	var results []SearchResult

	for _, fn := range s.index.Functions {
		if query.ExportedOnly && !fn.IsExported {
			continue
		}

		relevance := s.calculateFunctionRelevance(fn, queryTerms)
		if relevance > 0 {
			results = append(results, SearchResult{
				File:      fn.File,
				Line:      fn.Line,
				Type:      "function",
				Name:      fn.Name,
				Context:   s.formatFunctionContext(fn),
				Relevance: relevance,
				Keywords:  fn.Keywords,
			})
		}
	}

	return results
}

func (s *SemanticSearch) searchTypes(queryTerms []string, query SearchQuery) []SearchResult {
	var results []SearchResult

	for _, typ := range s.index.Types {
		if query.ExportedOnly && !typ.IsExported {
			continue
		}

		relevance := s.calculateTypeRelevance(typ, queryTerms)
		if relevance > 0 {
			results = append(results, SearchResult{
				File:      typ.File,
				Line:      typ.Line,
				Type:      "type",
				Name:      typ.Name,
				Context:   s.formatTypeContext(typ),
				Relevance: relevance,
				Keywords:  typ.Keywords,
			})
		}
	}

	return results
}

func (s *SemanticSearch) searchInterfaces(queryTerms []string, query SearchQuery) []SearchResult {
	var results []SearchResult

	for _, iface := range s.index.Interfaces {
		relevance := s.calculateInterfaceRelevance(iface, queryTerms)
		if relevance > 0 {
			results = append(results, SearchResult{
				File:      iface.File,
				Line:      0, // Interface line not tracked separately
				Type:      "interface",
				Name:      iface.Name,
				Context:   fmt.Sprintf("%s interface with %d methods", iface.Name, len(iface.Methods)),
				Relevance: relevance,
			})
		}
	}

	return results
}

func (s *SemanticSearch) searchComments(queryTerms []string, query SearchQuery) []SearchResult {
	var results []SearchResult

	for _, comment := range s.index.Comments {
		relevance := s.calculateCommentRelevance(comment, queryTerms)
		if relevance > 0 {
			results = append(results, SearchResult{
				File:      comment.File,
				Line:      comment.Line,
				Type:      "comment",
				Name:      comment.Type,
				Context:   s.truncateText(comment.Text, 100),
				Relevance: relevance,
				Keywords:  comment.Keywords,
			})
		}
	}

	return results
}

// Relevance calculation methods

func (s *SemanticSearch) calculateFunctionRelevance(fn FunctionInfo, queryTerms []string) float64 {
	score := 0.0

	// Name match (highest weight)
	nameTokens := s.tokenize(strings.ToLower(fn.Name))
	score += s.termMatchScore(nameTokens, queryTerms) * 3.0

	// Purpose match
	purposeTokens := s.tokenize(strings.ToLower(fn.Purpose))
	score += s.termMatchScore(purposeTokens, queryTerms) * 2.0

	// Doc comment match
	docTokens := s.tokenize(strings.ToLower(fn.DocComment))
	score += s.termMatchScore(docTokens, queryTerms) * 1.5

	// Keyword match
	keywordTokens := s.tokenizeSlice(fn.Keywords)
	score += s.termMatchScore(keywordTokens, queryTerms) * 1.0

	// Boost for exported functions
	if fn.IsExported {
		score *= 1.2
	}

	return score
}

func (s *SemanticSearch) calculateTypeRelevance(typ TypeInfo, queryTerms []string) float64 {
	score := 0.0

	nameTokens := s.tokenize(strings.ToLower(typ.Name))
	score += s.termMatchScore(nameTokens, queryTerms) * 3.0

	purposeTokens := s.tokenize(strings.ToLower(typ.Purpose))
	score += s.termMatchScore(purposeTokens, queryTerms) * 2.0

	docTokens := s.tokenize(strings.ToLower(typ.DocComment))
	score += s.termMatchScore(docTokens, queryTerms) * 1.5

	keywordTokens := s.tokenizeSlice(typ.Keywords)
	score += s.termMatchScore(keywordTokens, queryTerms) * 1.0

	// Boost based on type kind
	switch typ.Kind {
	case "interface":
		score *= 1.3
	case "struct":
		score *= 1.2
	}

	if typ.IsExported {
		score *= 1.2
	}

	return score
}

func (s *SemanticSearch) calculateInterfaceRelevance(iface InterfaceInfo, queryTerms []string) float64 {
	score := 0.0

	nameTokens := s.tokenize(strings.ToLower(iface.Name))
	score += s.termMatchScore(nameTokens, queryTerms) * 3.0

	// Method name matches
	for _, method := range iface.Methods {
		methodTokens := s.tokenize(strings.ToLower(method))
		score += s.termMatchScore(methodTokens, queryTerms) * 1.0
	}

	return score
}

func (s *SemanticSearch) calculateCommentRelevance(comment CommentInfo, queryTerms []string) float64 {
	textTokens := s.tokenize(strings.ToLower(comment.Text))
	score := s.termMatchScore(textTokens, queryTerms)

	// Boost important comment types
	switch comment.Type {
	case "todo", "fixme":
		score *= 1.3
	case "note":
		score *= 1.1
	}

	return score
}

// Helper methods

func (s *SemanticSearch) termMatchScore(tokens []string, queryTerms []string) float64 {
	if len(tokens) == 0 || len(queryTerms) == 0 {
		return 0.0
	}

	matches := 0
	for _, queryTerm := range queryTerms {
		for _, token := range tokens {
			if strings.Contains(token, queryTerm) {
				matches++
				break // Count each query term only once per token set
			}
		}
	}

	return float64(matches) / float64(len(queryTerms))
}

func (s *SemanticSearch) tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				token := current.String()
				if len(token) > 2 { // Skip very short tokens
					tokens = append(tokens, strings.ToLower(token))
				}
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		token := current.String()
		if len(token) > 2 {
			tokens = append(tokens, strings.ToLower(token))
		}
	}

	return tokens
}

func (s *SemanticSearch) tokenizeSlice(slice []string) []string {
	var tokens []string
	for _, item := range slice {
		tokens = append(tokens, s.tokenize(item)...)
	}
	return tokens
}

func (s *SemanticSearch) typeToString(expr ast.Expr) string {
	// Simplified type conversion - could be enhanced
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return s.typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + s.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + s.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + s.typeToString(t.Key) + "]" + s.typeToString(t.Value)
	default:
		return "interface{}"
	}
}

func (s *SemanticSearch) inferFilePurpose(filePath string, node *ast.File) string {
	baseName := filepath.Base(filePath)

	if strings.HasSuffix(baseName, "_test.go") {
		return "testing"
	}

	switch baseName {
	case "main.go":
		return "application entry point"
	case "doc.go":
		return "package documentation"
	}

	// Infer from package name
	pkg := node.Name.Name
	switch pkg {
	case "main":
		return "executable program"
	case "cmd":
		return "command line interface"
	default:
		return fmt.Sprintf("%s package implementation", pkg)
	}
}

func (s *SemanticSearch) extractFileKeywords(node *ast.File) []string {
	keywords := []string{node.Name.Name}

	// Add import keywords
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			keywords = append(keywords, parts[len(parts)-1])
		}
	}

	return s.dedupeStrings(keywords)
}

func (s *SemanticSearch) inferFunctionPurpose(fn FunctionInfo) string {
	name := strings.ToLower(fn.Name)

	// Common patterns
	if strings.HasPrefix(name, "new") {
		return "constructor"
	}
	if strings.HasPrefix(name, "get") {
		return "getter"
	}
	if strings.HasPrefix(name, "set") {
		return "setter"
	}
	if strings.HasPrefix(name, "is") || strings.HasPrefix(name, "has") {
		return "predicate"
	}
	if strings.HasPrefix(name, "test") {
		return "test"
	}
	if strings.Contains(name, "handler") {
		return "handler"
	}
	if strings.Contains(name, "parse") {
		return "parser"
	}
	if strings.Contains(name, "format") {
		return "formatter"
	}

	return "function"
}

func (s *SemanticSearch) extractFunctionKeywords(fn FunctionInfo) []string {
	keywords := []string{}

	// Add name tokens
	keywords = append(keywords, s.tokenize(fn.Name)...)

	// Add receiver type
	if fn.Receiver != "" {
		keywords = append(keywords, fn.Receiver)
	}

	// Add purpose
	keywords = append(keywords, fn.Purpose)

	return s.dedupeStrings(keywords)
}

func (s *SemanticSearch) inferTypePurpose(typ TypeInfo) string {
	name := strings.ToLower(typ.Name)

	if strings.HasSuffix(name, "config") {
		return "configuration"
	}
	if strings.HasSuffix(name, "request") {
		return "request data"
	}
	if strings.HasSuffix(name, "response") {
		return "response data"
	}
	if strings.HasSuffix(name, "error") {
		return "error type"
	}
	if strings.HasSuffix(name, "service") {
		return "service"
	}
	if strings.HasSuffix(name, "client") {
		return "client"
	}

	return typ.Kind
}

func (s *SemanticSearch) extractTypeKeywords(typ TypeInfo) []string {
	keywords := []string{}

	keywords = append(keywords, s.tokenize(typ.Name)...)
	keywords = append(keywords, typ.Kind)
	keywords = append(keywords, typ.Purpose)

	return s.dedupeStrings(keywords)
}

func (s *SemanticSearch) classifyComment(text string) string {
	lower := strings.ToLower(text)

	if strings.Contains(lower, "todo") {
		return "todo"
	}
	if strings.Contains(lower, "fixme") {
		return "fixme"
	}
	if strings.Contains(lower, "note:") {
		return "note"
	}
	if strings.Contains(lower, "warning") {
		return "warning"
	}
	if len(text) > 50 {
		return "doc"
	}

	return "comment"
}

func (s *SemanticSearch) extractCommentKeywords(text string) []string {
	// Simple keyword extraction from comments
	tokens := s.tokenize(text)

	// Filter out common words
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "this": true, "that": true,
		"with": true, "from": true, "are": true, "was": true, "will": true,
		"have": true, "has": true, "can": true, "could": true, "should": true,
	}

	var keywords []string
	for _, token := range tokens {
		if !commonWords[token] && len(token) > 3 {
			keywords = append(keywords, token)
		}
	}

	return s.dedupeStrings(keywords)
}

func (s *SemanticSearch) formatFunctionContext(fn FunctionInfo) string {
	context := fn.Name + "("
	if len(fn.Parameters) > 0 {
		context += strings.Join(fn.Parameters, ", ")
	}
	context += ")"

	if len(fn.ReturnTypes) > 0 {
		context += " " + strings.Join(fn.ReturnTypes, ", ")
	}

	if fn.Receiver != "" {
		context = "(" + fn.Receiver + ") " + context
	}

	return context
}

func (s *SemanticSearch) formatTypeContext(typ TypeInfo) string {
	context := fmt.Sprintf("%s %s", typ.Kind, typ.Name)

	if typ.Kind == "struct" && len(typ.Fields) > 0 {
		context += fmt.Sprintf(" with %d fields", len(typ.Fields))
	} else if typ.Kind == "interface" && len(typ.Methods) > 0 {
		context += fmt.Sprintf(" with %d methods", len(typ.Methods))
	}

	return context
}

func (s *SemanticSearch) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	truncated := text[:maxLen-3]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

func (s *SemanticSearch) dedupeStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Helper functions

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// SemanticSearchTool implements the Tool interface for CLI integration
type SemanticSearchTool struct {
	search *SemanticSearch
	logger logger.Logger
}

// NewSemanticSearchTool creates a new semantic search tool
func NewSemanticSearchTool(rootPath string) *SemanticSearchTool {
	return &SemanticSearchTool{
		search: NewSemanticSearch(rootPath),
		logger: logger.WithComponent("semantic-search-tool"),
	}
}

// Name returns the tool name
func (t *SemanticSearchTool) Name() string {
	return "semantic_search"
}

// Description returns the tool description
func (t *SemanticSearchTool) Description() string {
	return "Semantic search engine for code discovery and navigation"
}

// Execute runs semantic search (legacy interface)
func (t *SemanticSearchTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	action, ok := params["action"].(string)
	if !ok {
		action = "search"
	}

	switch action {
	case "build_index":
		return t.executeBuildIndex(ctx, params)
	case "search":
		return t.executeSearch(ctx, params)
	case "save_index":
		return t.executeSaveIndex(ctx, params)
	case "load_index":
		return t.executeLoadIndex(ctx, params)
	default:
		return "", nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (t *SemanticSearchTool) executeBuildIndex(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	if err := t.search.BuildIndex(ctx); err != nil {
		return "", nil, err
	}

	index := t.search.GetIndex()
	result := map[string]interface{}{
		"status":     "index_built",
		"files":      len(index.Files),
		"functions":  len(index.Functions),
		"types":      len(index.Types),
		"interfaces": len(index.Interfaces),
		"comments":   len(index.Comments),
		"keywords":   len(index.Keywords),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", nil, err
	}

	source := &models.EvidenceSource{
		Type:        "semantic_search",
		Resource:    "index_build",
		ExtractedAt: time.Now(),
		Metadata:    result,
	}

	return string(jsonData), source, nil
}

func (t *SemanticSearchTool) executeSearch(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	queryStr, ok := params["query"].(string)
	if !ok {
		return "", nil, fmt.Errorf("query parameter required")
	}

	query := SearchQuery{
		Query: queryStr,
		Limit: 20, // Default limit
	}

	// Parse optional parameters
	if types, ok := params["types"].([]string); ok {
		query.Types = types
	}
	if limit, ok := params["limit"].(int); ok {
		query.Limit = limit
	}
	if limitStr, ok := params["limit"].(string); ok {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = parsed
		}
	}
	if exported, ok := params["exported_only"].(bool); ok {
		query.ExportedOnly = exported
	}

	results, err := t.search.Search(ctx, query)
	if err != nil {
		return "", nil, err
	}

	response := map[string]interface{}{
		"query":   queryStr,
		"results": results,
		"count":   len(results),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", nil, err
	}

	source := &models.EvidenceSource{
		Type:        "semantic_search",
		Resource:    "search",
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"query":        queryStr,
			"result_count": len(results),
		},
	}

	return string(jsonData), source, nil
}

func (t *SemanticSearchTool) executeSaveIndex(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	path, ok := params["path"].(string)
	if !ok {
		path = ".ai-context/search-index.json"
	}

	if err := t.search.SaveIndex(path); err != nil {
		return "", nil, err
	}

	result := map[string]interface{}{
		"status": "index_saved",
		"path":   path,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", nil, err
	}

	source := &models.EvidenceSource{
		Type:        "semantic_search",
		Resource:    "save_index",
		ExtractedAt: time.Now(),
		Metadata:    result,
	}

	return string(jsonData), source, nil
}

func (t *SemanticSearchTool) executeLoadIndex(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	path, ok := params["path"].(string)
	if !ok {
		path = ".ai-context/search-index.json"
	}

	if err := t.search.LoadIndex(path); err != nil {
		return "", nil, err
	}

	result := map[string]interface{}{
		"status": "index_loaded",
		"path":   path,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", nil, err
	}

	source := &models.EvidenceSource{
		Type:        "semantic_search",
		Resource:    "load_index",
		ExtractedAt: time.Now(),
		Metadata:    result,
	}

	return string(jsonData), source, nil
}

// GetClaudeToolDefinition returns the Claude tool definition
func (t *SemanticSearchTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "semantic_search",
		Description: "Semantic search engine for code discovery",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "Action to perform",
					"enum":        []string{"build_index", "search", "save_index", "load_index"},
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (for search action)",
				},
				"types": map[string]interface{}{
					"type":        "array",
					"description": "Filter by types: function, type, interface, comment",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results",
				},
				"exported_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Only search exported symbols",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path for save/load operations",
				},
			},
		},
	}
}

// ExecuteTyped implements the TypedTool interface
func (t *SemanticSearchTool) ExecuteTyped(ctx context.Context, req types.Request) (types.Response, error) {
	// For now, delegate to the legacy Execute method
	adapter := types.NewToolAdapter(t)
	return adapter.ExecuteTyped(ctx, req)
}
