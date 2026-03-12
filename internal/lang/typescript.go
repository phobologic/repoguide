package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/tsx"

	"github.com/phobologic/repoguide/internal/model"
)

func init() {
	Languages["typescript"] = &Language{
		Name:              "typescript",
		Extensions:        []string{".js", ".jsx", ".ts", ".tsx", ".cjs", ".cts", ".mjs", ".mts"},
		lang:              tsx.GetLanguage(),
		FindReceiverType:  typescriptFindReceiverType,
		ExtractSignature:  typescriptExtractSignature,
		FindEnclosingDef:  typescriptFindEnclosingDef,
		FindEnclosingType: typescriptFindEnclosingType,
	}
}

func typescriptFindReceiverType(node *sitter.Node, source []byte) string {
	return typescriptFindEnclosingType(node, source)
}

func typescriptFindEnclosingType(node *sitter.Node, source []byte) string {
	current := node.Parent()
	for current != nil {
		switch current.Type() {
		case "class_declaration", "abstract_class_declaration", "class", "interface_declaration", "type_alias_declaration":
			return typescriptClassName(current, source)
		case "function_declaration", "generator_function_declaration", "function_expression", "generator_function", "arrow_function":
			return ""
		}
		current = current.Parent()
	}
	return ""
}

func typescriptFindEnclosingDef(node *sitter.Node, source []byte) string {
	current := node.Parent()
	for current != nil {
		switch current.Type() {
		case "function_declaration", "generator_function_declaration":
			return typescriptFunctionName(current, source)
		case "method_definition", "abstract_method_signature":
			methodName := typescriptMethodName(current, source)
			if methodName == "" {
				return ""
			}
			owner := typescriptFindEnclosingType(current, source)
			if owner == "" {
				return methodName
			}
			return owner + "." + methodName
		case "function_expression", "generator_function", "arrow_function":
			return ""
		}
		current = current.Parent()
	}
	return ""
}

func typescriptExtractSignature(defNode *sitter.Node, kind model.SymbolKind, source []byte) string {
	switch kind {
	case model.Class:
		return typescriptExtractTypeSignature(defNode, source)
	case model.Field:
		return typescriptExtractFieldSignature(defNode, source)
	default:
		return typescriptExtractFunctionSignature(defNode, source)
	}
}

func typescriptExtractTypeSignature(node *sitter.Node, source []byte) string {
	if body := node.ChildByFieldName("body"); body != nil {
		header := strings.TrimSpace(string(source[node.StartByte():body.StartByte()]))
		return CollapseWhitespace(header)
	}
	return CollapseWhitespace(NodeText(node, source))
}

func typescriptExtractFieldSignature(node *sitter.Node, source []byte) string {
	name := typescriptPropertyName(node.ChildByFieldName("name"), source)
	if name == "" {
		return CollapseWhitespace(NodeText(node, source))
	}

	annotation := node.ChildByFieldName("type")
	if annotation == nil {
		annotation = typescriptFirstChildOfType(node, "type_annotation")
	}
	if annotation == nil {
		return name
	}

	return name + CollapseWhitespace(NodeText(annotation, source))
}

func typescriptExtractFunctionSignature(node *sitter.Node, source []byte) string {
	name := typescriptFunctionLikeName(node, source)
	typeParameters := typescriptFieldText(node, "type_parameters", source)
	parameters := typescriptFieldText(node, "parameters", source)
	returnType := typescriptFieldText(node, "return_type", source)

	signature := name + typeParameters + parameters
	if returnType != "" {
		signature += " " + returnType
	}
	return signature
}

func typescriptFunctionLikeName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "function_declaration", "generator_function_declaration":
		return typescriptFunctionName(node, source)
	default:
		return typescriptMethodName(node, source)
	}
}

func typescriptFunctionName(node *sitter.Node, source []byte) string {
	return typescriptPropertyName(node.ChildByFieldName("name"), source)
}

func typescriptMethodName(node *sitter.Node, source []byte) string {
	return typescriptPropertyName(node.ChildByFieldName("name"), source)
}

func typescriptClassName(node *sitter.Node, source []byte) string {
	return typescriptPropertyName(node.ChildByFieldName("name"), source)
}

func typescriptPropertyName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	return NodeText(node, source)
}

func typescriptFieldText(node *sitter.Node, fieldName string, source []byte) string {
	child := node.ChildByFieldName(fieldName)
	if child == nil {
		return ""
	}
	return CollapseWhitespace(NodeText(child, source))
}

func typescriptFirstChildOfType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}
