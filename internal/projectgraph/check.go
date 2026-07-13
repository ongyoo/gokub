package projectgraph

import (
	"fmt"
	"sort"
	"strings"
)

type Violation struct {
	Type     string   `json:"type"`
	From     string   `json:"from,omitempty"`
	To       string   `json:"to,omitempty"`
	Packages []string `json:"packages,omitempty"`
	Message  string   `json:"message"`
}

type Analysis struct {
	OK         bool        `json:"ok"`
	Violations []Violation `json:"violations"`
	Graph      Graph       `json:"graph"`
}

func Analyze(graph Graph) Analysis {
	result := Analysis{OK: true, Violations: []Violation{}, Graph: graph}
	layers := map[string]string{}
	for _, node := range graph.Nodes {
		layers[node.ID] = node.Layer
	}
	for _, edge := range graph.Edges {
		fromLayer, fromOK := layers[edge.From]
		toLayer, toOK := layers[edge.To]
		if !fromOK || !toOK || !forbiddenBoundary(fromLayer, toLayer) {
			continue
		}
		result.Violations = append(result.Violations, Violation{
			Type: "boundary", From: edge.From, To: edge.To,
			Message: fmt.Sprintf("%s layer must not import %s layer", fromLayer, toLayer),
		})
	}
	for _, cycle := range stronglyConnected(graph) {
		result.Violations = append(result.Violations, Violation{
			Type: "cycle", Packages: cycle,
			Message: "internal package dependency cycle: " + strings.Join(cycle, " -> "),
		})
	}
	sort.Slice(result.Violations, func(i, j int) bool {
		left := result.Violations[i].Type + result.Violations[i].From + strings.Join(result.Violations[i].Packages, ",")
		right := result.Violations[j].Type + result.Violations[j].From + strings.Join(result.Violations[j].Packages, ",")
		return left < right
	})
	result.OK = len(result.Violations) == 0
	return result
}

func forbiddenBoundary(from, to string) bool {
	switch from {
	case "domain":
		return to == "platform" || to == "transport" || to == "service"
	case "platform":
		return to == "transport" || to == "service"
	case "transport":
		return to == "platform" || to == "service"
	case "shared":
		return to == "platform" || to == "transport" || to == "service"
	default:
		return false
	}
}

func stronglyConnected(graph Graph) [][]string {
	adjacency := map[string][]string{}
	known := map[string]bool{}
	for _, node := range graph.Nodes {
		known[node.ID] = true
	}
	for _, edge := range graph.Edges {
		if known[edge.From] && known[edge.To] {
			adjacency[edge.From] = append(adjacency[edge.From], edge.To)
		}
	}
	index := 0
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	stack := []string{}
	cycles := [][]string{}
	var visit func(string)
	visit = func(node string) {
		indices[node], lowlink[node] = index, index
		index++
		stack = append(stack, node)
		onStack[node] = true
		for _, next := range adjacency[node] {
			if _, seen := indices[next]; !seen {
				visit(next)
				lowlink[node] = min(lowlink[node], lowlink[next])
			} else if onStack[next] {
				lowlink[node] = min(lowlink[node], indices[next])
			}
		}
		if lowlink[node] != indices[node] {
			return
		}
		component := []string{}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == node {
				break
			}
		}
		selfCycle := len(component) == 1 && containsEdge(adjacency[component[0]], component[0])
		if len(component) > 1 || selfCycle {
			sort.Strings(component)
			cycles = append(cycles, component)
		}
	}
	for _, node := range graph.Nodes {
		if _, seen := indices[node.ID]; !seen {
			visit(node.ID)
		}
	}
	return cycles
}

func containsEdge(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
