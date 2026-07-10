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
			setWorkerCommand(process)
			addDevVolume(process)
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

// addDevVolume bind-mounts the director VM's /dev into each worker BPM
// container, replacing BPM's synthetic tmpfs /dev. Workers need
// /dev/loop-control and /dev/loopN to run "mount -o loop" during disk
// attachment.
//
// A plain mknod inside the container is not sufficient on Concourse workers:
// the host exposes ~257 pre-allocated loop devices (loop0–loop256) in the task
// container's /dev, so ioctl(LOOP_CTL_GET_FREE) returns a number >= 257.
// The corresponding node doesn't exist in BPM's synthetic /dev and mknod for
// it fails because the kernel's loop table is exhausted at the outer level.
// Bind-mounting /dev gives the worker the full set of pre-existing nodes so
// LOOP_CTL_GET_FREE always finds a matching device file.
//
// The function is idempotent: a second call on a process that already has /dev
// in its unrestricted_volumes is a no-op.
func addDevVolume(process *yaml.Node) {
	unsafeNode := findMappingValue(process, "unsafe")
	if unsafeNode == nil || unsafeNode.Kind != yaml.MappingNode {
		return
	}

	volsNode := findMappingValue(unsafeNode, "unrestricted_volumes")
	if volsNode != nil && volsNode.Kind == yaml.SequenceNode {
		for _, vol := range volsNode.Content {
			if vol.Kind == yaml.MappingNode {
				pathNode := findMappingValue(vol, "path")
				if pathNode != nil && pathNode.Value == "/dev" {
					return
				}
			}
		}
	}

	devEntry := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "path"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "/dev"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "writable"},
			{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		},
	}

	if volsNode == nil {
		unsafeNode.Content = append(unsafeNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "unrestricted_volumes"},
			&yaml.Node{
				Kind:    yaml.SequenceNode,
				Tag:     "!!seq",
				Content: []*yaml.Node{devEntry},
			},
		)
		return
	}

	volsNode.Content = append(volsNode.Content, devEntry)
}

// setMappingScalar sets an existing scalar key-value pair in a mapping node,
// or appends a new key-value pair if the key is not present. It is safe to
// call on any scalar-valued key regardless of the node's original tag or style.
func setMappingScalar(node *yaml.Node, key, value string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			v := node.Content[i+1]
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Value = value
			v.Content = nil
			return
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

// setWorkerCommand rewrites the executable and args of a worker process node so
// that the BPM container runs privileged (giving the CPI root for iptables/mount)
// while the worker process itself drops back to the vcap user via setpriv,
// preventing root-owned files from being written into /var/vcap/data/director.
// The original executable and any existing args are forwarded unchanged through
// the setpriv invocation. The transformation is idempotent — if the process
// already has /usr/bin/bash as its executable it has already been patched and
// is left as-is.
func setWorkerCommand(process *yaml.Node) {
	exeNode := findMappingValue(process, "executable")

	// Already patched — the executable has already been replaced with bash.
	if exeNode != nil && exeNode.Value == "/usr/bin/bash" {
		return
	}

	// Capture the original executable and any existing args so they can be
	// forwarded through the setpriv invocation unchanged.  Each token is
	// single-quote escaped so that values containing spaces or shell
	// metacharacters survive the bash -c evaluation intact.
	var parts []string
	parts = append(parts, "exec setpriv --reuid=vcap --regid=vcap --clear-groups --")
	if exeNode != nil && exeNode.Value != "" {
		parts = append(parts, shellQuote(exeNode.Value))
	}
	argsNode := findMappingValue(process, "args")
	if argsNode != nil && argsNode.Kind == yaml.SequenceNode {
		for _, arg := range argsNode.Content {
			if arg.Kind == yaml.ScalarNode {
				parts = append(parts, shellQuote(arg.Value))
			}
		}
	}

	execCmd := strings.Join(parts, " ")
	setMappingScalar(process, "executable", "/usr/bin/bash")

	newArgs := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "-c"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: execCmd},
		},
	}

	for i := 0; i+1 < len(process.Content); i += 2 {
		if process.Content[i].Value == "args" {
			process.Content[i+1] = newArgs
			return
		}
	}
	process.Content = append(process.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "args"},
		newArgs,
	)
}

// isWorkerProcess returns true for any director worker process that runs CPI
// calls: normal workers (worker_N) and dynamic disk workers (dynamic_disks_worker_N).
func isWorkerProcess(name string) bool {
	return strings.HasPrefix(name, "worker_") || strings.HasPrefix(name, "dynamic_disks_worker_")
}

// shellQuote wraps s in single quotes and escapes any embedded single quotes
// so the value survives evaluation by bash -c as a single word, regardless of
// spaces or shell metacharacters it may contain.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
