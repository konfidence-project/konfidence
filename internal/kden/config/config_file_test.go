package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeEach(func() {
	fileFuncs = fileFunctions{}
})

var _ = Describe("createConfigFile", func() {
	Context("when the config file does not exist", func() {

		It("should create the file", func() {
			tempDir := GinkgoT().TempDir()
			configFilePath = filepath.Join(tempDir, "kden", configFileName)
			err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			fileFuncs = fileFunctions{
				create: func(string) (string, error) {
					return configFilePath, nil
				},
				write: func(s string, bytes []byte, mode os.FileMode) error {
					return os.WriteFile(s, bytes, mode)
				},
			}
			configFilePath, err := createConfigFile()
			Expect(err).ToNot(HaveOccurred())
			Expect(configFilePath).ToNot(BeEmpty())
		})

		It("should return error if the file can't be created", func() {
			fileFuncs = fileFunctions{
				create: func(string) (string, error) {
					return "", os.ErrPermission
				},
			}
			_, err := createConfigFile()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create config directory"))
		})

		It("should return error if the file can't be written to", func() {
			fileFuncs = fileFunctions{
				create: func(string) (string, error) {
					return "", nil
				},
				write: func(s string, bytes []byte, mode os.FileMode) error {
					return os.ErrPermission
				},
			}
			_, err := createConfigFile()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create config file"))
		})
	})
})

var _ = Describe("getOrCreateConfigFile", func() {
	Context("when the config file does not exist", func() {

		It("should create the file and return its path", func() {
			tempDir := GinkgoT().TempDir()
			configFilePath = filepath.Join(tempDir, "kden", configFileName)
			err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			fileFuncs = fileFunctions{
				search: func(string) (string, error) {
					return "", os.ErrNotExist
				},
				create: func(string) (string, error) {
					return configFilePath, nil
				},
				write: func(s string, bytes []byte, mode os.FileMode) error {
					return os.WriteFile(s, bytes, mode)
				},
			}
			configFilePath, err := getOrCreateConfigFile()
			Expect(err).ToNot(HaveOccurred())
			Expect(configFilePath).ToNot(BeEmpty())
		})

		It("should return error if the file can't be created", func() {
			fileFuncs = fileFunctions{
				create: func(string) (string, error) {
					return "", os.ErrPermission
				},
				search: func(string) (string, error) {
					return "", os.ErrNotExist
				},
			}
			_, err := getOrCreateConfigFile()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create config directory"))
		})

		It("should return error if the file can't be written to", func() {
			fileFuncs = fileFunctions{
				create: func(string) (string, error) {
					return "", nil
				},
				search: func(string) (string, error) {
					return "", os.ErrNotExist
				},
				write: func(s string, bytes []byte, mode os.FileMode) error {
					return os.ErrPermission
				},
			}
			_, err := getOrCreateConfigFile()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create config file"))
		})

	})

	Context("when the config file already exists", func() {
		It("should return the file path", func() {
			fileFuncs = fileFunctions{
				search: func(string) (string, error) {
					return "somedir", nil
				},
			}

			configFilePath, err := getOrCreateConfigFile()
			Expect(err).ToNot(HaveOccurred())
			Expect(configFilePath).To(Equal("somedir"))
		})
	})
})

var _ = Describe("updateConfigFile", func() {
	Context("when writing to the file is successful", func() {
		It("should write the data to the file", func() {
			tempDir := GinkgoT().TempDir()
			configFilePath = filepath.Join(tempDir, "kden", configFileName)
			err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			fileFuncs = fileFunctions{
				write: func(s string, bytes []byte, mode os.FileMode) error {
					return os.WriteFile(s, bytes, mode)
				},
			}
			err = updateConfigFile(configFilePath, []byte(`{"key": "value"}`))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when writing to the file is not successful", func() {
		It("should throw an error when writing to the file", func() {
			fileFuncs = fileFunctions{
				write: func(s string, bytes []byte, mode os.FileMode) error {
					return os.ErrPermission
				},
			}
			err := updateConfigFile(configFilePath, []byte(`{"key": "value"}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to write config file"))
		})
	})
})
