package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// UpgradeNode is one normalized node of a validated upgrade tree.
type UpgradeNode struct {
	Key    string
	Depth  int16
	IsRoot bool
}

// parseUpgradeTree validates a definition upgrade tree and returns its
// normalized node projection. Depth is zero-based from the root and each node
// except the root has exactly one parent. Nodes are returned in deterministic
// key order so validation and persistence cannot disagree.
func parseUpgradeTree(tree json.RawMessage) ([]UpgradeNode, error) {
	var parsed struct {
		RootNodeKey string `json:"rootNodeKey"`
		Nodes       []struct {
			Key     string          `json:"key"`
			Name    string          `json:"name"`
			Next    []string        `json:"next"`
			Effects json.RawMessage `json:"effects"`
		} `json:"nodes"`
	}
	if len(tree) == 0 || !isJSONObject(tree) || json.Unmarshal(tree, &parsed) != nil {
		return nil, errors.New("upgrade tree must be a JSON object")
	}
	if !contentKeyPattern.MatchString(parsed.RootNodeKey) || len(parsed.Nodes) == 0 {
		return nil, errors.New("upgrade tree requires a valid rootNodeKey and nodes")
	}

	nextByKey := make(map[string][]string, len(parsed.Nodes))
	inDegree := make(map[string]int, len(parsed.Nodes))
	for _, node := range parsed.Nodes {
		if !contentKeyPattern.MatchString(node.Key) {
			return nil, fmt.Errorf("upgrade node %q has an invalid key", node.Key)
		}
		if node.Name == "" || !isJSONObject(node.Effects) {
			return nil, fmt.Errorf("upgrade node %q must have a name and object effects", node.Key)
		}
		if _, exists := nextByKey[node.Key]; exists {
			return nil, fmt.Errorf("duplicate upgrade node key %q", node.Key)
		}
		nextByKey[node.Key] = node.Next
		inDegree[node.Key] = 0
	}
	if _, exists := nextByKey[parsed.RootNodeKey]; !exists {
		return nil, fmt.Errorf("upgrade root %q is not a node", parsed.RootNodeKey)
	}
	for nodeKey, next := range nextByKey {
		for _, childKey := range next {
			if _, exists := nextByKey[childKey]; !exists {
				return nil, fmt.Errorf("upgrade node %q references unknown node %q", nodeKey, childKey)
			}
			inDegree[childKey]++
		}
	}
	if inDegree[parsed.RootNodeKey] != 0 {
		return nil, errors.New("upgrade root has an incoming edge")
	}
	for nodeKey, degree := range inDegree {
		if nodeKey != parsed.RootNodeKey && degree != 1 {
			return nil, fmt.Errorf("upgrade node %q must have exactly one parent", nodeKey)
		}
	}

	visiting := make(map[string]bool, len(nextByKey))
	visited := make(map[string]bool, len(nextByKey))
	var visit func(string) error
	visit = func(nodeKey string) error {
		if visiting[nodeKey] {
			return fmt.Errorf("upgrade tree contains cycle at %q", nodeKey)
		}
		if visited[nodeKey] {
			return nil
		}
		visiting[nodeKey] = true
		for _, childKey := range nextByKey[nodeKey] {
			if err := visit(childKey); err != nil {
				return err
			}
		}
		visiting[nodeKey] = false
		visited[nodeKey] = true
		return nil
	}
	if err := visit(parsed.RootNodeKey); err != nil {
		return nil, err
	}
	if len(visited) != len(nextByKey) {
		keys := make([]string, 0, len(nextByKey)-len(visited))
		for key := range nextByKey {
			if !visited[key] {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("upgrade tree is disconnected at %q", keys[0])
	}

	depthByKey := make(map[string]int16, len(nextByKey))
	depthByKey[parsed.RootNodeKey] = 0
	queue := []string{parsed.RootNodeKey}
	for head := 0; head < len(queue); head++ {
		for _, childKey := range nextByKey[queue[head]] {
			if _, seen := depthByKey[childKey]; seen {
				continue
			}
			depthByKey[childKey] = depthByKey[queue[head]] + 1
			queue = append(queue, childKey)
		}
	}

	nodes := make([]UpgradeNode, 0, len(nextByKey))
	for key := range nextByKey {
		nodes = append(nodes, UpgradeNode{
			Key:    key,
			Depth:  depthByKey[key],
			IsRoot: key == parsed.RootNodeKey,
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Key < nodes[j].Key })
	return nodes, nil
}
