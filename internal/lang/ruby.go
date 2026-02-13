package lang

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/ruby"

	"github.com/phobologic/repoguide/internal/model"
)

func init() {
	Languages["ruby"] = &Language{
		Name:             "ruby",
		Extensions:       []string{".rb"},
		lang:             ruby.GetLanguage(),
		FindMethodClass:  rubyFindMethodClass,
		ExtractSignature: rubyExtractSignature,
	}
}

// rubyFindMethodClass walks the parent chain looking for a class or module node.
func rubyFindMethodClass(funcNode *sitter.Node, source []byte) string {
	node := funcNode.Parent()
	for node != nil {
		if node.Type() == "class" || node.Type() == "module" {
			return rubyClassName(node, source)
		}
		node = node.Parent()
	}
	return ""
}

// rubyClassName extracts the name from a class or module node.
func rubyClassName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "constant" || child.Type() == "scope_resolution" {
			return NodeText(child, source)
		}
	}
	return ""
}

func rubyExtractSignature(defNode *sitter.Node, kind model.SymbolKind, source []byte) string {
	if kind == model.Class {
		return rubyExtractClassSignature(defNode, source)
	}
	return rubyExtractMethodSignature(defNode, source)
}

func rubyExtractClassSignature(node *sitter.Node, source []byte) string {
	var name, superclass string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "constant", "scope_resolution":
			if name == "" {
				name = NodeText(child, source)
			}
		case "superclass":
			// superclass node contains "< ClassName"
			for j := 0; j < int(child.ChildCount()); j++ {
				sc := child.Child(j)
				if sc.Type() == "constant" || sc.Type() == "scope_resolution" {
					superclass = NodeText(sc, source)
				}
			}
		}
	}
	if superclass != "" {
		return name + " < " + superclass
	}
	return name
}

func rubyExtractMethodSignature(node *sitter.Node, source []byte) string {
	var name, params string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			name = NodeText(child, source)
		case "method_parameters":
			params = CollapseWhitespace(NodeText(child, source))
		}
	}
	if params != "" {
		return name + params
	}
	return name
}
