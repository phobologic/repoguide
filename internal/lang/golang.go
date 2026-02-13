package lang

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"github.com/phobologic/repoguide/internal/model"
)

func init() {
	Languages["go"] = &Language{
		Name:             "go",
		Extensions:       []string{".go"},
		lang:             golang.GetLanguage(),
		FindReceiverType: goFindReceiverType,
		ExtractSignature: goExtractSignature,
	}
}

// goFindReceiverType extracts the receiver type name from a method_declaration node.
// Navigates: method_declaration → parameter_list (receiver) → parameter_declaration → type.
func goFindReceiverType(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "parameter_list" {
			continue
		}
		// The receiver is the first parameter_list (before the method name).
		// Check if a field_identifier follows this parameter_list.
		if !isReceiverList(node, child) {
			continue
		}
		for j := 0; j < int(child.ChildCount()); j++ {
			param := child.Child(j)
			if param.Type() == "parameter_declaration" {
				return goExtractTypeName(param, source)
			}
		}
	}
	return ""
}

// goExtractTypeName extracts the type name from a parameter_declaration,
// unwrapping pointer_type if present.
func goExtractTypeName(param *sitter.Node, source []byte) string {
	for i := 0; i < int(param.ChildCount()); i++ {
		child := param.Child(i)
		switch child.Type() {
		case "type_identifier":
			return NodeText(child, source)
		case "pointer_type":
			for k := 0; k < int(child.ChildCount()); k++ {
				inner := child.Child(k)
				if inner.Type() == "type_identifier" {
					return NodeText(inner, source)
				}
			}
		}
	}
	return ""
}

func goExtractSignature(defNode *sitter.Node, kind model.SymbolKind, source []byte) string {
	if kind == model.Class {
		// Type definition: just the type name
		for i := 0; i < int(defNode.ChildCount()); i++ {
			child := defNode.Child(i)
			if child.Type() == "type_identifier" {
				return NodeText(child, source)
			}
		}
		return ""
	}

	// Function or method
	var name, params, result string
	for i := 0; i < int(defNode.ChildCount()); i++ {
		child := defNode.Child(i)
		switch child.Type() {
		case "identifier", "field_identifier":
			name = NodeText(child, source)
		case "parameter_list":
			// For methods, the first parameter_list is the receiver — skip it
			if kind == model.Method && params == "" && isReceiverList(defNode, child) {
				continue
			}
			params = CollapseWhitespace(NodeText(child, source))
		case "simple_type", "pointer_type", "qualified_type",
			"slice_type", "map_type", "channel_type",
			"interface_type", "struct_type", "function_type",
			"type_identifier":
			result = CollapseWhitespace(NodeText(child, source))
		}
	}

	sig := name + params
	if result != "" {
		sig += " " + result
	}
	return sig
}

// isReceiverList checks if a parameter_list is the receiver (appears before the method name).
func isReceiverList(parent, paramList *sitter.Node) bool {
	if parent.Type() != "method_declaration" {
		return false
	}
	foundList := false
	for i := 0; i < int(parent.ChildCount()); i++ {
		child := parent.Child(i)
		if child == paramList {
			foundList = true
			continue
		}
		if foundList && child.Type() == "field_identifier" {
			return true
		}
	}
	return false
}
