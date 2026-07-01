package main

import (
	"fmt"
	"os"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

const (
	bpmYMLPath    = "/var/vcap/jobs/director/config/bpm.yml"
	headerComment = "THIS YAML HAS BEEN HACKILY MODIFIED BY THE WARDEN CPI"
)

func main() {
	path := bpmYMLPath
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	if err := patchBPMYML(path); err != nil {
		fmt.Fprintf(os.Stderr, "bpm-patcher: %v\n", err)
		os.Exit(1)
	}
}

func patchBPMYML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "bpm-patcher: %s not found, skipping\n", path)
			return nil
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected YAML structure in %s", path)
	}

	if doc.HeadComment == "" {
		doc.HeadComment = headerComment
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping at root of %s", path)
	}

	processesNode := findMappingValue(root, "processes")
	if processesNode == nil || processesNode.Kind != yaml.SequenceNode {
		fmt.Fprintf(os.Stderr, "bpm-patcher: could not find 'processes' sequence in %s, skipping\n", path)
		return nil
	}

	patched := 0
	for _, process := range processesNode.Content {
		if process.Kind != yaml.MappingNode {
			continue
		}
		nameNode := findMappingValue(process, "name")
		if nameNode == nil {
			continue
		}
		if isWorkerProcess(nameNode.Value) {
			setPrivileged(process)
			patched++
		}
	}

	fmt.Printf("bpm-patcher: patched %d worker process(es) in %s\n", patched, path)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if err := os.WriteFile(path, out, info.Mode()); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// isWorkerProcess returns true for any director worker process that runs CPI
// calls: normal workers (worker_N) and dynamic disk workers (dynamic_disks_worker_N).
func isWorkerProcess(name string) bool {
	return strings.HasPrefix(name, "worker_") || strings.HasPrefix(name, "dynamic_disks_worker_")
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// setPrivileged adds unsafe.privileged: true to a BPM process node.
// It is idempotent — running it twice on the same node is safe.
func setPrivileged(process *yaml.Node) {
	unsafeNode := findMappingValue(process, "unsafe")
	if unsafeNode == nil {
		process.Content = append(process.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "unsafe"},
			&yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "privileged"},
					{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
				},
			},
		)
		return
	}

	// If the unsafe value exists but is not a mapping (unexpected schema),
	// replace it with a fresh mapping containing just privileged: true.
	if unsafeNode.Kind != yaml.MappingNode {
		unsafeNode.Kind = yaml.MappingNode
		unsafeNode.Tag = ""
		unsafeNode.Value = ""
		unsafeNode.Content = []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "privileged"},
			{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		}
		return
	}

	privilegedNode := findMappingValue(unsafeNode, "privileged")
	if privilegedNode == nil {
		// Prepend so privileged: true is the first key in the unsafe block.
		unsafeNode.Content = append(
			[]*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "privileged"},
				{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
			},
			unsafeNode.Content...,
		)
	} else {
		privilegedNode.Value = "true"
		privilegedNode.Tag = "!!bool"
	}
}
