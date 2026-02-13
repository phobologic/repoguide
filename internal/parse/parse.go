// Package parse extracts tags from source files using tree-sitter.
package parse

import (
	"context"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/phobologic/repoguide/internal/model"
)

var captureMap = map[string]struct {
	Kind       model.TagKind
	SymbolKind model.SymbolKind
}{
	"definition.class":    {model.Definition, model.Class},
	"definition.function": {model.Definition, model.Function},
	"reference.call":      {model.Reference, model.Function},
	"reference.import":    {model.Reference, model.Module},
}

var whitespaceRe = regexp.MustCompile(`\s+`)

// ExtractTags parses a source file and returns definition and reference tags.
// The parser must be created for the correct language.
// filePath is used only for Tag.File and should be the repo-relative path.
func ExtractTags(parser *sitter.Parser, query *sitter.Query, source []byte, filePath string) []model.Tag {
	if len(source) == 0 {
		return nil
	}

	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil
	}
	defer tree.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(query, tree.RootNode())

	var tags []model.Tag

	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}
		match = qc.FilterPredicates(match, source)

		// Find the @name capture and the pattern capture
		var nameNode *sitter.Node
		var captureName string
		var defNode *sitter.Node

		for _, c := range match.Captures {
			cname := query.CaptureNameForId(c.Index)
			if cname == "name" {
				nameNode = c.Node
			} else if _, ok := captureMap[cname]; ok {
				captureName = cname
				defNode = c.Node
			}
		}

		if nameNode == nil || captureName == "" || defNode == nil {
			continue
		}

		cm := captureMap[captureName]
		tagKind := cm.Kind
		symbolKind := cm.SymbolKind
		nameText := nodeText(nameNode, source)

		effectiveName := nameText

		if tagKind == model.Definition && symbolKind == model.Function && isMethod(defNode) {
			symbolKind = model.Method
			if className := getEnclosingClassName(defNode, source); className != "" {
				effectiveName = className + "." + nameText
			}
		}

		var signature string
		if tagKind == model.Definition {
			signature = extractSignature(defNode, symbolKind, source)
		}

		tags = append(tags, model.Tag{
			Name:       effectiveName,
			Kind:       tagKind,
			SymbolKind: symbolKind,
			Line:       int(nameNode.StartPoint().Row) + 1,
			File:       filePath,
			Signature:  signature,
		})
	}

	return tags
}

func nodeText(node *sitter.Node, source []byte) string {
	return string(source[node.StartByte():node.EndByte()])
}

func findEnclosingClass(funcNode *sitter.Node) *sitter.Node {
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

func getEnclosingClassName(funcNode *sitter.Node, source []byte) string {
	classNode := findEnclosingClass(funcNode)
	if classNode == nil {
		return ""
	}
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "identifier" {
			return nodeText(child, source)
		}
	}
	return ""
}

func isMethod(funcNode *sitter.Node) bool {
	return findEnclosingClass(funcNode) != nil
}

func extractSignature(defNode *sitter.Node, symbolKind model.SymbolKind, source []byte) string {
	if symbolKind == model.Class {
		return extractClassSignature(defNode, source)
	}
	return extractFunctionSignature(defNode, source)
}

func extractClassSignature(node *sitter.Node, source []byte) string {
	var name, args string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			name = nodeText(child, source)
		case "argument_list":
			args = nodeText(child, source)
		}
	}
	if args != "" {
		return name + args
	}
	return name
}

func extractFunctionSignature(node *sitter.Node, source []byte) string {
	var name, params, returnType string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			name = nodeText(child, source)
		case "parameters":
			params = collapseWhitespace(nodeText(child, source))
		case "type":
			returnType = nodeText(child, source)
		}
	}
	sig := name + params
	if returnType != "" {
		sig += " -> " + returnType
	}
	return sig
}

func collapseWhitespace(s string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(s, " "))
}
