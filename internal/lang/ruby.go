package lang

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/ruby"

	"github.com/phobologic/repoguide/internal/model"
)

func init() {
	Languages["ruby"] = &Language{
		Name:              "ruby",
		Extensions:        []string{".rb"},
		lang:              ruby.GetLanguage(),
		FindMethodClass:   rubyFindMethodClass,
		ExtractSignature:  rubyExtractSignature,
		FindEnclosingDef:  rubyFindEnclosingDef,
		FindEnclosingType: rubyFindEnclosingType,
	}
}

// rubyFindEnclosingDef returns the qualified name of the method containing
// the given call-site node (e.g., "MyClass.method" or "methodName").
// Returns "" if the call is at class/module body level or script top-level.
func rubyFindEnclosingDef(node *sitter.Node, source []byte) string {
	current := node.Parent()
	for current != nil {
		switch current.Type() {
		case "method":
			var methodName string
			for i := 0; i < int(current.ChildCount()); i++ {
				child := current.Child(i)
				if child.Type() == "identifier" {
					methodName = NodeText(child, source)
					break
				}
			}
			if methodName == "" {
				return ""
			}
			// Walk further ancestors for enclosing class/module.
			ancestor := current.Parent()
			for ancestor != nil {
				if ancestor.Type() == "class" || ancestor.Type() == "module" {
					cls := rubyClassName(ancestor, source)
					if cls != "" {
						return cls + "." + methodName
					}
					break
				}
				ancestor = ancestor.Parent()
			}
			return methodName

		case "singleton_method":
			// def self.foo — find the last identifier (the method name, not "self").
			var methodName string
			for i := 0; i < int(current.ChildCount()); i++ {
				child := current.Child(i)
				if child.Type() == "identifier" {
					methodName = NodeText(child, source)
				}
			}
			if methodName == "" {
				return ""
			}
			ancestor := current.Parent()
			for ancestor != nil {
				if ancestor.Type() == "class" || ancestor.Type() == "module" {
					cls := rubyClassName(ancestor, source)
					if cls != "" {
						return cls + "." + methodName
					}
					break
				}
				ancestor = ancestor.Parent()
			}
			return methodName
		}
		current = current.Parent()
	}
	return ""
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

// rubyFindEnclosingType walks up from a call node (attr_accessor etc.) to find
// the enclosing class or module name. Returns "" if not inside a class/module.
func rubyFindEnclosingType(node *sitter.Node, source []byte) string {
	current := node.Parent()
	for current != nil {
		if current.Type() == "class" || current.Type() == "module" {
			return rubyClassName(current, source)
		}
		current = current.Parent()
	}
	return ""
}

func rubyExtractSignature(defNode *sitter.Node, kind model.SymbolKind, source []byte) string {
	if kind == model.Class {
		return rubyExtractClassSignature(defNode, source)
	}
	if kind == model.Field {
		return rubyExtractFieldSignature(defNode, source)
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

// rubyExtractFieldSignature returns the signature for an attr_accessor/reader/writer
// field. The defNode is the call expression; we extract the method name to form
// e.g. "attr_accessor :field_name".
func rubyExtractFieldSignature(defNode *sitter.Node, source []byte) string {
	var methodName string
	for i := 0; i < int(defNode.ChildCount()); i++ {
		child := defNode.Child(i)
		if child.Type() == "identifier" {
			methodName = NodeText(child, source)
			break
		}
	}
	return methodName
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
