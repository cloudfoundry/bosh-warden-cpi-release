package main

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	yaml "go.yaml.in/yaml/v3"
)

// minimalBPMYML is a representative director bpm.yml with worker and
// non-worker processes, and an existing unsafe block on one worker.
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
		It("sets unsafe.privileged on every worker process", func() {
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
				} else {
					// non-worker processes must not gain an unsafe block
					if unsafe != nil {
						priv := findMappingValue(unsafe, "privileged")
						Expect(priv).To(BeNil(), "director must not be privileged")
					}
				}
			}
		})

		It("preserves existing unsafe keys on a worker", func() {
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
			Expect(findMappingValue(unsafe, "unrestricted_volumes")).NotTo(BeNil(),
				"pre-existing unsafe keys must be preserved")
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
