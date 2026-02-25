---
id: repoguide-bpa.7
status: closed
deps: [repoguide-bpa.1]
links: []
created: 2026-02-13T07:25:31.068776-08:00
type: task
priority: 2
parent: repoguide-bpa
---
# Implement dependency graph + PageRank (internal/graph)

Port graph.py to Go. BuildGraph(files []FileInfo) (*Graph, []Dependency). Graph as adjacency map: map[string]map[string]int (from→to→edge_count). Build defines map (symbol→set of files), then for each reference tag, add edges to definition files (skip self-edges). Aggregate symbols per (src,tgt) pair into Dependency objects. Hand-rolled iterative PageRank: alpha=0.85, max 100 iterations, 1e-6 tolerance. Mutates FileInfo.Rank in place, sorts descending. Tests: pre-built FileInfo fixtures, verify edges, no self-edges, PageRank sum≈1.0, more-referenced file ranks higher, empty graph gives uniform.


