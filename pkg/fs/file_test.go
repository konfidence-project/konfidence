package fs_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/fs"
)

var _ = Describe("FileData", func() {

	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "file-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
	})

	DescribeTable("SetFilePath with no errors",
		func(filename, input, wantFile string) {
			file := filepath.Join(tmpDir, filename)
			Expect(os.WriteFile(file, []byte("key: value"), 0644)).To(Succeed())

			fd := &fs.FileData{}
			Expect(fd.SetFilePath(filepath.Join(tmpDir, input))).To(Succeed())
			Expect(fd.GetFilePath()).To(Equal(filepath.Join(tmpDir, wantFile)))
		},
		Entry("resolves .yaml when given path without extension",
			"component.yaml", "component", "component.yaml"),
		Entry("resolves .yml when given path without extension",
			"component.yml", "component", "component.yml"),
		Entry("accepts path that already has .yaml extension",
			"component.yaml", "component.yaml", "component.yaml"),
		Entry("accepts path that already has .yml extension",
			"component.yml", "component.yml", "component.yml"),
	)

	Describe("SetFilePath", func() {
		It("returns an error when neither .yaml nor .yml exists", func() {
			fd := &fs.FileData{}
			err := fd.SetFilePath(filepath.Join(tmpDir, "nonexistent"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nonexistent"))
			Expect(err.Error()).To(ContainSubstring(".yaml/.yml"))
		})
	})

	Describe("ReadFile", func() {
		It("returns the file contents", func() {
			content := []byte("name: my-component\nversion: v1.0.0\n")
			yamlFile := filepath.Join(tmpDir, "component.yaml")
			Expect(os.WriteFile(yamlFile, content, 0644)).To(Succeed())

			fd := &fs.FileData{}
			Expect(fd.SetFilePath(yamlFile)).To(Succeed())

			data, err := fs.ReadFile(fd)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(content))
		})

		It("returns an error when the FileData has no file", func() {
			fd := &fs.FileData{}
			data, err := fs.ReadFile(fd)
			Expect(err).To(HaveOccurred())
			Expect(data).To(BeNil())
		})
	})

	Describe("ToFileData", func() {
		It("converts a list of paths to FileData slices", func() {
			pathA := filepath.Join(tmpDir, "a.yaml")
			pathB := filepath.Join(tmpDir, "b.yml")
			Expect(os.WriteFile(pathA, []byte("a: 1"), 0644)).To(Succeed())
			Expect(os.WriteFile(pathB, []byte("b: 2"), 0644)).To(Succeed())

			result, err := fs.ToFileData([]string{pathA, pathB})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].GetFilePath()).To(Equal(pathA))
			Expect(result[1].GetFilePath()).To(Equal(pathB))
		})

		It("returns an error when a path does not exist", func() {
			result, err := fs.ToFileData([]string{filepath.Join(tmpDir, "missing")})
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("returns an error when a path is a directory", func() {
			subDir := filepath.Join(tmpDir, "subdir")
			Expect(os.Mkdir(subDir, 0755)).To(Succeed())
			// ToFileData calls SetFilePath, which requires .yaml/.yml - create a yaml-named dir workaround
			// by creating a dir with .yaml suffix
			yamlDir := filepath.Join(tmpDir, "component.yaml")
			Expect(os.Mkdir(yamlDir, 0755)).To(Succeed())

			result, err := fs.ToFileData([]string{yamlDir})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("directory"))
			Expect(result).To(BeNil())
		})

		It("returns an empty slice for an empty input", func() {
			result, err := fs.ToFileData([]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("stops at the first error and returns nil", func() {
			validFile := filepath.Join(tmpDir, "valid.yaml")
			Expect(os.WriteFile(validFile, []byte("a: 1"), 0644)).To(Succeed())

			result, err := fs.ToFileData([]string{validFile, filepath.Join(tmpDir, "missing")})
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
