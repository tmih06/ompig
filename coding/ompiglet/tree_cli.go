package ompiglet

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

// TreeNode is one Ompiglet in the extends tree. Installed means a built Ompiglet
// Binary is recorded for it; Current means the environment selects it
// (OMPIG_OMPIGLET_NAME or OMPIG_OMPIGLET_PATH). External marks a base that is named by
// an `extends` but is not in the inventory.
// ompig additive (D170): `ompig ompiglet tree` reports the inventory `ompig ompiglet list` reads, arranged by `extends`.
type TreeNode struct {
	Name      string      `json:"name"`
	Version   string      `json:"version,omitempty"`
	Location  string      `json:"location,omitempty"`
	Source    string      `json:"source,omitempty"`
	Installed bool        `json:"installed"`
	Current   bool        `json:"current"`
	External  bool        `json:"external,omitempty"`
	Children  []*TreeNode `json:"children"`
}

const treeUsage = "Usage: ompig ompiglet tree [name] [--depth <n>] [--json]"

func cmdTree(args []string, stdout, stderr io.Writer) int {
	jsonMode, name, depth := false, "", -1
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--json":
			jsonMode = true
		case arg == "--depth" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				_, _ = fmt.Fprintf(stderr, "error: --depth needs a non-negative number\n%s\n", treeUsage)
				return 2
			}
			depth = n
		case arg == "-h" || arg == "--help":
			_, _ = fmt.Fprintln(stdout, treeUsage)
			return 0
		case strings.HasPrefix(arg, "-") || name != "":
			_, _ = fmt.Fprintln(stderr, treeUsage)
			return 2
		default:
			name = arg
		}
	}
	inventory, err := List()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: Ompiglet inventory is invalid: %v\n", err)
		return 1
	}
	roots, err := BuildTree(inventory, os.Getenv("OMPIG_OMPIGLET_NAME"), os.Getenv("OMPIG_OMPIGLET_PATH"))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if name != "" {
		node := findNode(roots, name)
		if node == nil {
			_, _ = fmt.Fprintf(stderr, "error: no Ompiglet %q in the inventory\n", name)
			return 1
		}
		roots = []*TreeNode{node}
	}
	for _, root := range roots {
		limitDepth(root, depth)
	}
	if jsonMode {
		//portlint:allow all Ompig-only `ompig ompiglet tree --json` output with no Pi counterpart (Pi has no Ompiglets); the map has one key, so its order cannot differ
		data, _ := json.MarshalIndent(map[string]any{"ompiglets": roots}, "", "  ")
		_, _ = fmt.Fprintln(stdout, string(data))
		return 0
	}
	RenderTree(stdout, roots)
	return 0
}

// BuildTree arranges the inventory by `extends`, offline: it reads the source
// files the inventory names. currentName and currentPath are the environment's
// selection.
func BuildTree(inventory []PigletInfo, currentName, currentPath string) ([]*TreeNode, error) {
	if currentPath != "" {
		if p, err := Parse(currentPath); err == nil {
			currentName = p.Name
		}
	}
	nodes := map[string]*TreeNode{}
	parentOf := map[string]string{}
	for _, info := range inventory {
		node := &TreeNode{Name: info.Name, Location: info.Location, Source: info.Path, Installed: len(info.Records) > 0, Current: info.Name == currentName && currentName != "", Children: []*TreeNode{}}
		for _, record := range info.Records {
			if node.Version == "" {
				node.Version = record.ReleaseVersion
			}
		}
		if info.Path != "" {
			links, err := sourceChain(info.Path)
			if err != nil {
				return nil, fmt.Errorf("Ompiglet %s: %w", info.Name, err)
			}
			leaf := links[len(links)-1]
			if leaf.ompiglet.Release != nil {
				node.Version = leaf.ompiglet.Release.Version
			}
			if len(links) > 1 {
				parent := links[len(links)-2]
				parentOf[info.Name] = parent.ompiglet.Name
				if _, ok := nodes[parent.ompiglet.Name]; !ok {
					base := &TreeNode{Name: parent.ompiglet.Name, Source: parent.path, External: true, Children: []*TreeNode{}}
					if parent.ompiglet.Release != nil {
						base.Version = parent.ompiglet.Release.Version
					}
					nodes[parent.ompiglet.Name] = base
				}
			}
		}
		if existing, ok := nodes[info.Name]; ok && existing.External {
			node.Children = existing.Children
		}
		nodes[info.Name] = node
	}
	var roots []*TreeNode
	for name, node := range nodes {
		if parent, ok := parentOf[name]; ok && nodes[parent] != nil && parent != name {
			nodes[parent].Children = append(nodes[parent].Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	byName := func(a, b *TreeNode) int { return strings.Compare(a.Name, b.Name) }
	slices.SortFunc(roots, byName)
	for _, node := range nodes {
		slices.SortFunc(node.Children, byName)
	}
	return roots, nil
}

func findNode(nodes []*TreeNode, name string) *TreeNode {
	for _, node := range nodes {
		if node.Name == name {
			return node
		}
		if found := findNode(node.Children, name); found != nil {
			return found
		}
	}
	return nil
}

// limitDepth drops the nodes more than depth levels below root; a negative depth keeps all.
func limitDepth(node *TreeNode, depth int) {
	if depth < 0 {
		return
	}
	if depth == 0 {
		node.Children = []*TreeNode{}
		return
	}
	for _, child := range node.Children {
		limitDepth(child, depth-1)
	}
}

// RenderTree draws roots with box-drawing characters. `*` marks the current
// Ompiglet, `[installed]` one with a built Binary, `(external)` a base outside
// the inventory.
func RenderTree(w io.Writer, roots []*TreeNode) {
	for _, root := range roots {
		_, _ = fmt.Fprintln(w, treeLabel(root))
		renderChildren(w, root.Children, "")
	}
}

func renderChildren(w io.Writer, children []*TreeNode, prefix string) {
	for i, child := range children {
		branch, next := "├── ", "│   "
		if i == len(children)-1 {
			branch, next = "└── ", "    "
		}
		_, _ = fmt.Fprintln(w, prefix+branch+treeLabel(child))
		renderChildren(w, child.Children, prefix+next)
	}
}

func treeLabel(n *TreeNode) string {
	label := n.Name
	if n.Version != "" {
		label += " " + n.Version
	}
	if n.Current {
		label = "* " + label
	}
	if n.Installed {
		label += " [installed]"
	}
	if n.External {
		label += " (external)"
	}
	return label
}
