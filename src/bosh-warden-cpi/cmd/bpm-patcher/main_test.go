package main

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	yaml "go.yaml.in/yaml/v3"
)

// minimalBPMYML is a representative director bpm.yml with worker and
// non-worker processes, an existing unsafe block on one worker, and a worker
// that already has an executable and args (to exercise replacement logic).
const minimalBPMYML = `processes:
- name: director
  executable: /var/vcap/packages/director/bin/bosh-director
- name: worker_0
  executable: /var/vcap/packages/director/bin/bosh-director-worker
- name: worker_1
  executable: /var/vcap/packages/director/bin/bosh-director-worker
- name: dynamic_disks_worker_0
  executable: /var/vcap/packages/director/bin/bosh-director-worker
- name: worker_with_existing_unsafe
  executable: /var/vcap/packages/director/bin/bosh-director-worker
  unsafe:
    unrestricted_volumes:
    - path: /var/vcap/data
- name: worker_with_existing_args
  executable: /some/old/executable
  args:
  - --old-arg
  - --another-old-arg
`

func writeTempBPM(dir, content string) string {
	path := filepath.Join(dir, "bpm.yml")
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}

func parseBPM(path string) *yaml.Node {
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	var doc yaml.Node
	Expect(yaml.Unmarshal(data, &doc)).To(Succeed())
	return &doc
}

func buildProcess(extraContent ...*yaml.Node) *yaml.Node {
	base := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "name"},
		{Kind: yaml.ScalarNode, Value: "worker_0"},
		{Kind: yaml.ScalarNode, Value: "executable"},
		{Kind: yaml.ScalarNode, Value: "/path/to/binary"},
	}
	return &yaml.Node{Kind: yaml.MappingNode, Content: append(base, extraContent...)}
}

var _ = Describe("patchBPMYML", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "bpm-patcher-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("when the file does not exist", func() {
		It("succeeds without error", func() {
			err := patchBPMYML(filepath.Join(tmpDir, "nonexistent.yml"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when the YAML root is not a mapping", func() {
		It("returns an error", func() {
			path := writeTempBPM(tmpDir, "- item1\n- item2\n")
			err := patchBPMYML(path)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected mapping at root"))
		})
	})

	Context("when 'processes' is absent", func() {
		It("succeeds without error (soft-skip)", func() {
			path := writeTempBPM(tmpDir, "some_other_key: value\n")
			Expect(patchBPMYML(path)).To(Succeed())
		})
	})

	Context("when 'processes' is not a sequence", func() {
		It("succeeds without error (soft-skip)", func() {
			path := writeTempBPM(tmpDir, "processes: not-a-list\n")
			Expect(patchBPMYML(path)).To(Succeed())
		})
	})

	Context("with a valid bpm.yml", func() {
		It("sets unsafe.privileged, setpriv command, and /dev volume on every worker process", func() {
			path := writeTempBPM(tmpDir, minimalBPMYML)
			Expect(patchBPMYML(path)).To(Succeed())

			doc := parseBPM(path)
			root := doc.Content[0]
			processes := findMappingValue(root, "processes")
			Expect(processes).NotTo(BeNil())

			for _, proc := range processes.Content {
				name := findMappingValue(proc, "name")
				Expect(name).NotTo(BeNil())

				unsafe := findMappingValue(proc, "unsafe")
				if isWorkerProcess(name.Value) {
					Expect(unsafe).NotTo(BeNil(), "expected unsafe block on %s", name.Value)
					priv := findMappingValue(unsafe, "privileged")
					Expect(priv).NotTo(BeNil(), "expected privileged key on %s", name.Value)
					Expect(priv.Value).To(Equal("true"))

					exe := findMappingValue(proc, "executable")
					Expect(exe).NotTo(BeNil(), "expected executable on %s", name.Value)
					Expect(exe.Value).To(Equal("/usr/bin/bash"), "executable on %s", name.Value)

					argsNode := findMappingValue(proc, "args")
					Expect(argsNode).NotTo(BeNil(), "expected args on %s", name.Value)
					Expect(argsNode.Kind).To(Equal(yaml.SequenceNode))
					Expect(argsNode.Content).To(HaveLen(2))
					Expect(argsNode.Content[0].Value).To(Equal("-c"))
					Expect(argsNode.Content[1].Value).To(ContainSubstring("setpriv"))

					vols := findMappingValue(unsafe, "unrestricted_volumes")
					Expect(vols).NotTo(BeNil(), "expected unrestricted_volumes on %s", name.Value)
					lastVol := vols.Content[len(vols.Content)-1]
					Expect(findMappingValue(lastVol, "path").Value).To(Equal("/dev"),
						"/dev volume missing on %s", name.Value)
					Expect(findMappingValue(lastVol, "writable").Value).To(Equal("true"),
						"writable missing on %s", name.Value)
				} else {
					// non-worker processes must not gain an unsafe block or altered executable
					if unsafe != nil {
						priv := findMappingValue(unsafe, "privileged")
						Expect(priv).To(BeNil(), "director must not be privileged")
					}
					exe := findMappingValue(proc, "executable")
					Expect(exe).NotTo(BeNil())
					Expect(exe.Value).NotTo(Equal("/usr/bin/bash"), "director executable must not be rewritten")
				}
			}
		})

		It("preserves the original executable and args inside the setpriv command", func() {
			path := writeTempBPM(tmpDir, minimalBPMYML)
			Expect(patchBPMYML(path)).To(Succeed())

			doc := parseBPM(path)
			root := doc.Content[0]
			processes := findMappingValue(root, "processes")

			var target *yaml.Node
			for _, proc := range processes.Content {
				name := findMappingValue(proc, "name")
				if name != nil && name.Value == "worker_with_existing_args" {
					target = proc
					break
				}
			}
			Expect(target).NotTo(BeNil())

			argsNode := findMappingValue(target, "args")
			Expect(argsNode).NotTo(BeNil())
			cmd := argsNode.Content[1].Value
			Expect(cmd).To(ContainSubstring("setpriv"))
			Expect(cmd).To(ContainSubstring("/some/old/executable"))
			Expect(cmd).To(ContainSubstring("--old-arg"))
			Expect(cmd).To(ContainSubstring("--another-old-arg"))
		})

		It("preserves existing unsafe keys on a worker and adds /dev volume alongside them", func() {
			path := writeTempBPM(tmpDir, minimalBPMYML)
			Expect(patchBPMYML(path)).To(Succeed())

			doc := parseBPM(path)
			root := doc.Content[0]
			processes := findMappingValue(root, "processes")

			var workerWithExisting *yaml.Node
			for _, proc := range processes.Content {
				name := findMappingValue(proc, "name")
				if name != nil && name.Value == "worker_with_existing_unsafe" {
					workerWithExisting = proc
					break
				}
			}
			Expect(workerWithExisting).NotTo(BeNil())

			unsafe := findMappingValue(workerWithExisting, "unsafe")
			Expect(unsafe).NotTo(BeNil())
			Expect(findMappingValue(unsafe, "privileged").Value).To(Equal("true"))

			volsNode := findMappingValue(unsafe, "unrestricted_volumes")
			Expect(volsNode).NotTo(BeNil(), "pre-existing unrestricted_volumes must be preserved")
			Expect(volsNode.Kind).To(Equal(yaml.SequenceNode))
			Expect(volsNode.Content).To(HaveLen(2))

			firstPath := findMappingValue(volsNode.Content[0], "path")
			Expect(firstPath.Value).To(Equal("/var/vcap/data"))
			secondPath := findMappingValue(volsNode.Content[1], "path")
			Expect(secondPath.Value).To(Equal("/dev"))
			secondWritable := findMappingValue(volsNode.Content[1], "writable")
			Expect(secondWritable.Value).To(Equal("true"))
		})

		It("adds the header comment", func() {
			path := writeTempBPM(tmpDir, minimalBPMYML)
			Expect(patchBPMYML(path)).To(Succeed())

			data, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(headerComment))
		})

		It("is idempotent — a second call does not double-add unsafe", func() {
			path := writeTempBPM(tmpDir, minimalBPMYML)
			Expect(patchBPMYML(path)).To(Succeed())
			firstContent, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(patchBPMYML(path)).To(Succeed())
			secondContent, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(secondContent)).To(Equal(string(firstContent)))
		})
	})
})

var _ = Describe("isWorkerProcess", func() {
	DescribeTable("worker names",
		func(name string, expected bool) {
			Expect(isWorkerProcess(name)).To(Equal(expected))
		},
		Entry("worker_0", "worker_0", true),
		Entry("worker_99", "worker_99", true),
		Entry("dynamic_disks_worker_0", "dynamic_disks_worker_0", true),
		Entry("dynamic_disks_worker_1", "dynamic_disks_worker_1", true),
		Entry("director", "director", false),
		Entry("health_monitor", "health_monitor", false),
		Entry("worker (no underscore suffix)", "worker", false),
		Entry("dynamic_disks_worker (no suffix)", "dynamic_disks_worker", false),
	)
})

var _ = Describe("findMappingValue", func() {
	It("returns the value node for a present key", func() {
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "foo"},
				{Kind: yaml.ScalarNode, Value: "bar"},
			},
		}
		result := findMappingValue(node, "foo")
		Expect(result).NotTo(BeNil())
		Expect(result.Value).To(Equal("bar"))
	})

	It("returns nil for a missing key", func() {
		node := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		Expect(findMappingValue(node, "missing")).To(BeNil())
	})

	It("returns nil for a non-mapping node", func() {
		node := &yaml.Node{Kind: yaml.SequenceNode}
		Expect(findMappingValue(node, "foo")).To(BeNil())
	})
})

var _ = Describe("setWorkerCommand", func() {
	buildProcess := func(extraContent ...*yaml.Node) *yaml.Node {
		base := []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name"},
			{Kind: yaml.ScalarNode, Value: "worker_0"},
			{Kind: yaml.ScalarNode, Value: "executable"},
			{Kind: yaml.ScalarNode, Value: "/var/vcap/packages/director/bin/bosh-director-worker"},
		}
		return &yaml.Node{Kind: yaml.MappingNode, Content: append(base, extraContent...)}
	}

	It("sets executable to /usr/bin/bash", func() {
		proc := buildProcess()
		setWorkerCommand(proc)
		exe := findMappingValue(proc, "executable")
		Expect(exe).NotTo(BeNil())
		Expect(exe.Value).To(Equal("/usr/bin/bash"))
	})

	It("wraps the original executable in a setpriv command", func() {
		proc := buildProcess()
		setWorkerCommand(proc)
		args := findMappingValue(proc, "args")
		Expect(args).NotTo(BeNil())
		Expect(args.Kind).To(Equal(yaml.SequenceNode))
		Expect(args.Content).To(HaveLen(2))
		Expect(args.Content[0].Value).To(Equal("-c"))
		Expect(args.Content[1].Value).To(ContainSubstring("setpriv"))
		Expect(args.Content[1].Value).To(ContainSubstring("/var/vcap/packages/director/bin/bosh-director-worker"))
	})

	Context("process has existing args", func() {
		It("forwards existing args through the setpriv command", func() {
			proc := buildProcess(
				&yaml.Node{Kind: yaml.ScalarNode, Value: "args"},
				&yaml.Node{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "worker_0"},
						{Kind: yaml.ScalarNode, Value: "--extra-flag"},
					},
				},
			)
			setWorkerCommand(proc)

			args := findMappingValue(proc, "args")
			Expect(args.Content).To(HaveLen(2))
			cmd := args.Content[1].Value
			Expect(cmd).To(ContainSubstring("setpriv"))
			Expect(cmd).To(ContainSubstring("/var/vcap/packages/director/bin/bosh-director-worker"))
			Expect(cmd).To(ContainSubstring("worker_0"))
			Expect(cmd).To(ContainSubstring("--extra-flag"))
		})
	})

	Context("process has an arg that contains spaces", func() {
		It("shell-quotes the arg so it survives bash -c as a single word", func() {
			proc := buildProcess(
				&yaml.Node{Kind: yaml.ScalarNode, Value: "args"},
				&yaml.Node{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "hello world"},
					},
				},
			)
			setWorkerCommand(proc)

			args := findMappingValue(proc, "args")
			Expect(args.Content).To(HaveLen(2))
			cmd := args.Content[1].Value
			// The space-containing arg must appear quoted so bash treats it as one token.
			Expect(cmd).To(ContainSubstring("'hello world'"))
		})
	})

	Context("called twice on the same process", func() {
		It("is idempotent — does not wrap setpriv inside itself", func() {
			proc := buildProcess()
			setWorkerCommand(proc)
			firstCmd := findMappingValue(proc, "args").Content[1].Value

			setWorkerCommand(proc)
			args := findMappingValue(proc, "args")
			Expect(args.Content).To(HaveLen(2), "args must not be duplicated")
			Expect(args.Content[1].Value).To(Equal(firstCmd),
				"second call must not re-wrap the setpriv command")
		})
	})
})

var _ = Describe("setMappingScalar", func() {
	It("adds a new key when absent", func() {
		node := &yaml.Node{Kind: yaml.MappingNode}
		setMappingScalar(node, "foo", "bar")
		result := findMappingValue(node, "foo")
		Expect(result).NotTo(BeNil())
		Expect(result.Value).To(Equal("bar"))
	})

	It("updates an existing key in place", func() {
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "foo"},
				{Kind: yaml.ScalarNode, Value: "old"},
			},
		}
		setMappingScalar(node, "foo", "new")
		result := findMappingValue(node, "foo")
		Expect(result.Value).To(Equal("new"))
		Expect(node.Content).To(HaveLen(2), "must not append a duplicate key")
	})
})

var _ = Describe("setPrivileged", func() {
	buildProcess := func(extraContent ...*yaml.Node) *yaml.Node {
		base := []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name"},
			{Kind: yaml.ScalarNode, Value: "worker_0"},
			{Kind: yaml.ScalarNode, Value: "executable"},
			{Kind: yaml.ScalarNode, Value: "/path/to/binary"},
		}
		return &yaml.Node{Kind: yaml.MappingNode, Content: append(base, extraContent...)}
	}

	Context("process has no unsafe block", func() {
		It("adds unsafe.privileged: true", func() {
			proc := buildProcess()
			setPrivileged(proc)

			unsafe := findMappingValue(proc, "unsafe")
			Expect(unsafe).NotTo(BeNil())
			Expect(findMappingValue(unsafe, "privileged").Value).To(Equal("true"))
		})
	})

	Context("process has unsafe block without privileged key", func() {
		It("prepends privileged: true to the existing unsafe block", func() {
			proc := buildProcess(
				&yaml.Node{Kind: yaml.ScalarNode, Value: "unsafe"},
				&yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "unrestricted_volumes"},
						{Kind: yaml.SequenceNode},
					},
				},
			)
			setPrivileged(proc)

			unsafe := findMappingValue(proc, "unsafe")
			Expect(findMappingValue(unsafe, "privileged").Value).To(Equal("true"))
			Expect(findMappingValue(unsafe, "unrestricted_volumes")).NotTo(BeNil(),
				"existing keys must be preserved")
		})
	})

	Context("process has unsafe.privileged: false", func() {
		It("sets it to true", func() {
			proc := buildProcess(
				&yaml.Node{Kind: yaml.ScalarNode, Value: "unsafe"},
				&yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "privileged"},
						{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"},
					},
				},
			)
			setPrivileged(proc)

			unsafe := findMappingValue(proc, "unsafe")
			Expect(findMappingValue(unsafe, "privileged").Value).To(Equal("true"))
		})
	})

	Context("called twice on the same process", func() {
		It("is idempotent", func() {
			proc := buildProcess()
			setPrivileged(proc)
			setPrivileged(proc)

			unsafe := findMappingValue(proc, "unsafe")
			Expect(unsafe).NotTo(BeNil())
			Expect(findMappingValue(unsafe, "privileged").Value).To(Equal("true"))
			// unsafe block should only have the one key
			Expect(unsafe.Content).To(HaveLen(2))
		})
	})

	Context("process has a malformed unsafe block (not a mapping)", func() {
		It("replaces it with a valid unsafe.privileged: true mapping", func() {
			proc := buildProcess(
				&yaml.Node{Kind: yaml.ScalarNode, Value: "unsafe"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: "some-unexpected-scalar"},
			)
			setPrivileged(proc)

			unsafe := findMappingValue(proc, "unsafe")
			Expect(unsafe).NotTo(BeNil())
			Expect(unsafe.Kind).To(Equal(yaml.MappingNode))
			Expect(findMappingValue(unsafe, "privileged").Value).To(Equal("true"))
		})
	})
})

var _ = Describe("addDevVolume", func() {
	buildSafeProcess := func(extraPairs ...*yaml.Node) *yaml.Node {
		// Build a process node that already has a valid unsafe block (with
		// privileged: true) so addDevVolume has a node to attach volumes to.
		unsafeContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "privileged"},
			{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		}
		unsafeContent = append(unsafeContent, extraPairs...)
		unsafe := &yaml.Node{Kind: yaml.MappingNode, Content: unsafeContent}
		return buildProcess(
			&yaml.Node{Kind: yaml.ScalarNode, Value: "unsafe"},
			unsafe,
		)
	}

	Context("process has an unsafe block without unrestricted_volumes", func() {
		It("adds a /dev unrestricted_volume with writable: true", func() {
			proc := buildSafeProcess()
			unsafe := findMappingValue(proc, "unsafe")
			Expect(unsafe).NotTo(BeNil())

			addDevVolume(proc)

			vols := findMappingValue(unsafe, "unrestricted_volumes")
			Expect(vols).NotTo(BeNil())
			Expect(vols.Content).To(HaveLen(1))
			Expect(findMappingValue(vols.Content[0], "path").Value).To(Equal("/dev"))
			Expect(findMappingValue(vols.Content[0], "writable").Value).To(Equal("true"))
		})
	})

	Context("process has an unsafe block with existing unrestricted_volumes", func() {
		It("appends /dev without removing existing volumes", func() {
			proc := buildSafeProcess(
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "unrestricted_volumes"},
				&yaml.Node{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Value: "path"},
								{Kind: yaml.ScalarNode, Value: "/var/vcap/data"},
							},
						},
					},
				},
			)
			unsafe := findMappingValue(proc, "unsafe")
			addDevVolume(proc)

			vols := findMappingValue(unsafe, "unrestricted_volumes")
			Expect(vols).NotTo(BeNil())
			Expect(vols.Content).To(HaveLen(2))
			Expect(findMappingValue(vols.Content[0], "path").Value).To(Equal("/var/vcap/data"))
			Expect(findMappingValue(vols.Content[1], "path").Value).To(Equal("/dev"))
			Expect(findMappingValue(vols.Content[1], "writable").Value).To(Equal("true"))
		})
	})

	Context("called twice on the same process", func() {
		It("is idempotent — does not add /dev twice", func() {
			proc := buildSafeProcess()
			unsafe := findMappingValue(proc, "unsafe")

			addDevVolume(proc)
			addDevVolume(proc)

			vols := findMappingValue(unsafe, "unrestricted_volumes")
			Expect(vols).NotTo(BeNil())
			Expect(vols.Content).To(HaveLen(1), "should only have one /dev entry")
		})
	})

	Context("process has no unsafe block", func() {
		It("does nothing gracefully", func() {
			proc := &yaml.Node{Kind: yaml.MappingNode}
			Expect(func() { addDevVolume(proc) }).NotTo(Panic())
			Expect(findMappingValue(proc, "unsafe")).To(BeNil())
		})
	})
})
