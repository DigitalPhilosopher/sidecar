package filebrowser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTree_Build(t *testing.T) {
	// Use current directory for testing
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tree := NewFileTree(cwd)
	if err := tree.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tree.Len() == 0 {
		t.Error("Expected non-empty tree")
	}

	// Verify root is set
	if tree.Root == nil {
		t.Error("Expected root to be set")
	}
}

func TestFileTree_ExpandCollapse(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "subdir", "nested"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("test"), 0644)

	tree := NewFileTree(tmpDir)
	if err := tree.Build(); err != nil {
		t.Fatal(err)
	}

	// Find the subdir node
	var subdirNode *FileNode
	for _, node := range tree.FlatList {
		if node.Name == "subdir" && node.IsDir {
			subdirNode = node
			break
		}
	}

	if subdirNode == nil {
		t.Fatal("Expected to find subdir node")
	}

	initialLen := tree.Len()

	// Expand
	if err := tree.Expand(subdirNode); err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if tree.Len() <= initialLen {
		t.Error("Expected tree length to increase after expand")
	}

	expandedLen := tree.Len()

	// Collapse
	tree.Collapse(subdirNode)

	if tree.Len() >= expandedLen {
		t.Error("Expected tree length to decrease after collapse")
	}
}

func TestFileTree_GetNode(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)

	tree := NewFileTree(tmpDir)
	if err := tree.Build(); err != nil {
		t.Fatal(err)
	}

	// Valid index
	node := tree.GetNode(0)
	if node == nil {
		t.Error("Expected node at index 0")
	}

	// Invalid indices
	if tree.GetNode(-1) != nil {
		t.Error("Expected nil for negative index")
	}
	if tree.GetNode(1000) != nil {
		t.Error("Expected nil for out of bounds index")
	}
}

func TestSortChildren(t *testing.T) {
	children := []*FileNode{
		{Name: "zebra.txt", IsDir: false},
		{Name: "alpha", IsDir: true},
		{Name: "beta.txt", IsDir: false},
		{Name: "delta", IsDir: true},
	}

	sortChildren(children)

	// Directories should come first
	if !children[0].IsDir || !children[1].IsDir {
		t.Error("Directories should be sorted first")
	}

	// Then alphabetical
	if children[0].Name != "alpha" || children[1].Name != "delta" {
		t.Error("Directories should be alphabetically sorted")
	}
	if children[2].Name != "beta.txt" || children[3].Name != "zebra.txt" {
		t.Error("Files should be alphabetically sorted")
	}
}
