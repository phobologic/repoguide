package lang

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"

	"github.com/phobologic/repoguide/internal/model"
)

func init() {
	Languages["python"] = &Language{
		Name:              "python",
		Extensions:        []string{".py"},
		lang:              python.GetLanguage(),
		FindMethodClass:   pythonFindMethodClass,
		ExtractSignature:  pythonExtractSignature,
		FindEnclosingDef:  pythonFindEnclosingDef,
		FindEnclosingType: pythonFindEnclosingType,
	}
}

// pythonFindEnclosingDef returns the qualified name of the function or method
// containing the given call-site node (e.g., "MyClass.method" or "funcName").
// Returns "" if the call is at module top-level.
func pythonFindEnclosingDef(node *sitter.Node, source []byte) string {
	current := node.Parent()
	for current != nil {
		if current.Type() == "function_definition" {
			var funcName string
			for i := 0; i < int(current.ChildCount()); i++ {
				child := current.Child(i)
				if child.Type() == "identifier" {
					funcName = NodeText(child, source)
					break
				}
			}
			if funcName == "" {
				return ""
			}
			if cls := pythonFindEnclosingClass(current); cls != nil {
				for i := 0; i < int(cls.ChildCount()); i++ {
					child := cls.Child(i)
					if child.Type() == "identifier" {
						return NodeText(child, source) + "." + funcName
					}
				}
			}
			return funcName
		}
		current = current.Parent()
	}
	return ""
}

func pythonFindMethodClass(funcNode *sitter.Node, source []byte) string {
	classNode := pythonFindEnclosingClass(funcNode)
	if classNode == nil {
		return ""
	}
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "identifier" {
			return NodeText(child, source)
		}
	}
	return ""
}

func pythonFindEnclosingClass(funcNode *sitter.Node) *sitter.Node {
	parent := funcNode.Parent()
	if parent == nil {
		return nil
	}

	// Direct: func -> block -> class_definition
	if parent.Type() == "block" && parent.Parent() != nil && parent.Parent().Type() == "class_definition" {
		return parent.Parent()
	}

	// Decorated: func -> decorated_definition -> block -> class_definition
	if parent.Type() == "decorated_definition" {
		gp := parent.Parent()
		if gp != nil && gp.Type() == "block" && gp.Parent() != nil && gp.Parent().Type() == "class_definition" {
			return gp.Parent()
		}
	}

	return nil
}

func pythonExtractSignature(defNode *sitter.Node, kind model.SymbolKind, source []byte) string {
	if kind == model.Class {
		return pythonExtractClassSignature(defNode, source)
	}
	if kind == model.Field {
		return pythonExtractFieldSignature(defNode, source)
	}
	return pythonExtractFunctionSignature(defNode, source)
}

func pythonExtractClassSignature(node *sitter.Node, source []byte) string {
	var name, args string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			name = NodeText(child, source)
		case "argument_list":
			args = NodeText(child, source)
		}
	}
	if args != "" {
		return name + args
	}
	return name
}

// pythonFindEnclosingType walks up from a field node to find the enclosing
// class_definition and returns its name. Returns "" if not inside a class,
// or if inside a function/method body (not a class-level attribute).
func pythonFindEnclosingType(node *sitter.Node, source []byte) string {
	current := node.Parent()
	for current != nil {
		switch current.Type() {
		case "class_definition":
			for i := 0; i < int(current.ChildCount()); i++ {
				child := current.Child(i)
				if child.Type() == "identifier" {
					return NodeText(child, source)
				}
			}
		case "function_definition":
			// Inside a method body — not a class-level attribute.
			return ""
		}
		current = current.Parent()
	}
	return ""
}

// pythonExtractFieldSignature extracts a signature for a class field.
// In this grammar version, annotated assignments (x: Type = val) are represented
// as an assignment node with a "type" child. Returns "name: type" when a type
// annotation is present, otherwise just "name".
func pythonExtractFieldSignature(node *sitter.Node, source []byte) string {
	if node.Type() == "assignment" {
		var name, annotation string
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "identifier":
				if name == "" {
					name = NodeText(child, source)
				}
			case "type":
				annotation = NodeText(child, source)
			}
		}
		if name != "" && annotation != "" {
			return name + ": " + annotation
		}
		if name != "" {
			return name
		}
	}
	return CollapseWhitespace(NodeText(node, source))
}

func pythonExtractFunctionSignature(node *sitter.Node, source []byte) string {
	var name, params, returnType string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			name = NodeText(child, source)
		case "parameters":
			params = CollapseWhitespace(NodeText(child, source))
		case "type":
			returnType = NodeText(child, source)
		}
	}
	sig := name + params
	if returnType != "" {
		sig += " -> " + returnType
	}
	return sig
}
