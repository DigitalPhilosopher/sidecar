package docdrift

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// CodeFeature represents a feature extracted from code.
type CodeFeature struct {
	Name        string // Function, type, interface name
	Type        string // "function", "type", "interface", "method"
	Signature   string // Full signature for functions
	Package     string // Package name
	SourceFile  string // Path to source file
	IsExported  bool   // Is it publicly exported?
}

// CodeAnalyzer extracts features from Go code.
type CodeAnalyzer struct {
	RootDir string
	Features []CodeFeature
}

// NewCodeAnalyzer creates a new code analyzer.
func NewCodeAnalyzer(rootDir string) *CodeAnalyzer {
	return &CodeAnalyzer{
		RootDir:  rootDir,
		Features: []CodeFeature{},
	}
}

// AnalyzePackage extracts exported items from a Go package.
func (ca *CodeAnalyzer) AnalyzePackage(packagePath string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.AllErrors)
	if err != nil {
		return err
	}

	for pkgName, pkg := range pkgs {
		for fileName, f := range pkg.Files {
			ca.analyzeFile(f, pkgName, fileName)
		}
	}

	return nil
}

func (ca *CodeAnalyzer) analyzeFile(f *ast.File, pkgName, fileName string) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			ca.analyzeGenDecl(d, pkgName, fileName)
		case *ast.FuncDecl:
			ca.analyzeFuncDecl(d, pkgName, fileName)
		}
	}
}

func (ca *CodeAnalyzer) analyzeGenDecl(d *ast.GenDecl, pkgName, fileName string) {
	switch d.Tok {
	case token.TYPE:
		for _, spec := range d.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok && ast.IsExported(typeSpec.Name.Name) {
				ca.Features = append(ca.Features, CodeFeature{
					Name:       typeSpec.Name.Name,
					Type:       "type",
					Package:    pkgName,
					SourceFile: filepath.Base(fileName),
					IsExported: true,
				})
			}
		}
	case token.CONST, token.VAR:
		for _, spec := range d.Specs {
			if valSpec, ok := spec.(*ast.ValueSpec); ok {
				for _, name := range valSpec.Names {
					if ast.IsExported(name.Name) {
						ca.Features = append(ca.Features, CodeFeature{
							Name:       name.Name,
							Type:       strings.ToLower(d.Tok.String()),
							Package:    pkgName,
							SourceFile: filepath.Base(fileName),
							IsExported: true,
						})
					}
				}
			}
		}
	}
}

func (ca *CodeAnalyzer) analyzeFuncDecl(d *ast.FuncDecl, pkgName, fileName string) {
	if !ast.IsExported(d.Name.Name) {
		return
	}

	featureType := "function"
	if d.Recv != nil {
		featureType = "method"
	}

	ca.Features = append(ca.Features, CodeFeature{
		Name:       d.Name.Name,
		Type:       featureType,
		Signature:  d.Name.Name + "()",
		Package:    pkgName,
		SourceFile: filepath.Base(fileName),
		IsExported: true,
	})
}

// ExtractPluginNames extracts plugin IDs from the plugins directory.
func (ca *CodeAnalyzer) ExtractPluginNames() ([]string, error) {
	pluginDir := filepath.Join(ca.RootDir, "internal", "plugins")
	entries, err := filepath.Glob(pluginDir + "/*")
	if err != nil {
		return nil, err
	}

	var plugins []string
	for _, entry := range entries {
		// Only include directories that are actual plugins
		name := filepath.Base(entry)
		// Skip hidden files
		if !strings.HasPrefix(name, ".") {
			plugins = append(plugins, name)
		}
	}

	return plugins, nil
}
