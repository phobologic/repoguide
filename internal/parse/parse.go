// Package parse extracts tags from source files using tree-sitter.
package parse

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/phobologic/repoguide/internal/lang"
	"github.com/phobologic/repoguide/internal/model"
)

var captureMap = map[string]struct {
	Kind       model.TagKind
	SymbolKind model.SymbolKind
}{
	"definition.class":    {model.Definition, model.Class},
	"definition.function": {model.Definition, model.Function},
	"definition.method":   {model.Definition, model.Method},
	"reference.call":      {model.Reference, model.Function},
	"reference.import":    {model.Reference, model.Module},
}

// ExtractTags parses a source file and returns definition and reference tags.
// The parser must be created for the correct language.
// filePath is used only for Tag.File and should be the repo-relative path.
func ExtractTags(l *lang.Language, parser *sitter.Parser, query *sitter.Query, source []byte, filePath string) []model.Tag {
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
		nameText := lang.NodeText(nameNode, source)

		effectiveName := nameText

		// Go-style: query captured @definition.method directly
		if tagKind == model.Definition && symbolKind == model.Method {
			if l.FindReceiverType != nil {
				if recv := l.FindReceiverType(defNode, source); recv != "" {
					effectiveName = recv + "." + nameText
				}
			}
		} else if tagKind == model.Definition && symbolKind == model.Function {
			// Python/Ruby-style: check if function is inside a class
			if l.FindMethodClass != nil {
				if cls := l.FindMethodClass(defNode, source); cls != "" {
					symbolKind = model.Method
					effectiveName = cls + "." + nameText
				}
			}
		}

		var signature string
		if tagKind == model.Definition && l.ExtractSignature != nil {
			signature = l.ExtractSignature(defNode, symbolKind, source)
		}

		var enclosing string
		if tagKind == model.Reference && symbolKind == model.Function {
			if l.FindEnclosingDef != nil {
				enclosing = l.FindEnclosingDef(defNode, source)
			}
		}

		tags = append(tags, model.Tag{
			Name:       effectiveName,
			Kind:       tagKind,
			SymbolKind: symbolKind,
			Line:       int(nameNode.StartPoint().Row) + 1,
			File:       filePath,
			Signature:  signature,
			Enclosing:  enclosing,
		})
	}

	return tags
}
