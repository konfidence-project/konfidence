package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var cmd *cobra.Command
var configFilePath string
var jsonLiteral = "json"

var (
	fileLogLevel  = "info"
	fileLogFormat = jsonLiteral
	fileOutput    = jsonLiteral

	envLogLevel  = "debug"
	envLogFormat = "text"
	envOutput    = jsonLiteral

	flagLogLevel  = "error"
	flagLogFormat = "pretty"
	flagOutput    = "pretty"
)

var _ = BeforeEach(func() {
	tempDir := GinkgoT().TempDir()
	configFilePath = filepath.Join(tempDir, "kden", configFileName)
	err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm)
	Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(configFilePath, []byte("{}"), 0644)
	Expect(err).ToNot(HaveOccurred())

	envConfigs = envVarConfigs{
		kdenEnvPrefix:    "CONFIG_TEST_KDEN_",
		logLevelEnvName:  "CONFIG_TEST_KDEN_LOG_LEVEL",
		logFormatEnvName: "CONFIG_TEST_KDEN_LOG_FORMAT",
		outputEnvName:    "CONFIG_TEST_KDEN_OUTPUT",
	}

	configFileFuncs = configFileFunctions{
		getPath: func() (string, error) {
			return configFilePath, nil
		},
		updateFile: func(path string, data []byte) error {
			return os.WriteFile(path, data, 0644)
		},
	}

	cmd = &cobra.Command{}
	cmd.Flags().String("log-level", "", "The level at which to log output.")
	cmd.Flags().String("log-format", "", "The level at which to log output.")
	cmd.Flags().String("output", "", "The output format for the logs.")

	_ = os.Unsetenv(envConfigs.logLevelEnvName)
	_ = os.Unsetenv(envConfigs.logFormatEnvName)
	_ = os.Unsetenv(envConfigs.outputEnvName)

	xdg.Reload()
})

var _ = Describe("Configure", func() {
	Context("with default configuration", func() {
		It("should return log-level 'error', log-format 'pretty', output `json`", func() {
			err := Configure(cmd)
			Expect(err).ToNot(HaveOccurred())
			Expect(Config.LogLevel).To(Equal("error"))
			Expect(Config.LogFormat).To(Equal("pretty"))
			Expect(Config.Output).To(Equal(jsonLiteral))
		})
	})

	Context("with file configuration", func() {
		It("should return envVarConfigs from file", func() {
			setupConfigurationFile()
			err := Configure(cmd)
			Expect(err).ToNot(HaveOccurred())
			Expect(Config.LogLevel).To(Equal(fileLogLevel))
			Expect(Config.LogFormat).To(Equal(fileLogFormat))
			Expect(Config.Output).To(Equal(fileOutput))
		})
	})

	Context("with KDEN_LOG_LEVEL, KDEN_LOG_FORMAT, KDEN_OUTPUT_FORMAT environment variables specified", func() {
		It("should return configuration from environment variables", func() {
			err := os.Setenv(envConfigs.logLevelEnvName, envLogLevel)
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.logFormatEnvName, envLogFormat)
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.outputEnvName, envOutput)
			Expect(err).ToNot(HaveOccurred())

			err = Configure(cmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(Config.LogLevel).To(Equal(envLogLevel))
			Expect(Config.LogFormat).To(Equal(envLogFormat))
			Expect(Config.Output).To(Equal(envOutput))

		})
	})

	Context("with only '--log-level', '--output' and '--log-format' flags specified", func() {
		It("should return envVarConfigs from flag", func() {
			err := cmd.Flags().Set("log-level", flagLogLevel)
			Expect(err).ToNot(HaveOccurred())

			err = cmd.Flags().Set("log-format", flagLogFormat)
			Expect(err).ToNot(HaveOccurred())

			err = cmd.Flags().Set("output", flagOutput)
			Expect(err).ToNot(HaveOccurred())

			err = Configure(cmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(Config.LogLevel).To(Equal(flagLogLevel))
			Expect(Config.LogFormat).To(Equal(flagLogFormat))
			Expect(Config.Output).To(Equal(flagOutput))

		})
	})

	Context("with all possible envVarConfigs specified", func() {
		It("should return flag configurations", func() {
			setupConfigurationFile()
			err := cmd.Flags().Set("log-level", flagLogLevel)
			Expect(err).ToNot(HaveOccurred())

			err = cmd.Flags().Set("log-format", flagLogFormat)
			Expect(err).ToNot(HaveOccurred())

			err = cmd.Flags().Set("output", flagOutput)
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.logLevelEnvName, envLogLevel)
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.logFormatEnvName, envLogFormat)
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.outputEnvName, envOutput)
			Expect(err).ToNot(HaveOccurred())

			err = Configure(cmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(Config.LogLevel).To(Equal(flagLogLevel))
			Expect(Config.LogFormat).To(Equal(flagLogFormat))
			Expect(Config.Output).To(Equal(flagOutput))
		})
	})

	Context("with invalid environment variables specified", func() {
		It("should return default values", func() {
			err := os.Setenv(envConfigs.kdenEnvPrefix+"my-log-level", "env-level")
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.kdenEnvPrefix+"output-*-format", "env-level")
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(envConfigs.kdenEnvPrefix+"log-*-format", "env-level")
			Expect(err).ToNot(HaveOccurred())

			err = Configure(cmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(Config.LogLevel).To(Equal("error"))
			Expect(Config.LogFormat).To(Equal("pretty"))
			Expect(Config.Output).To(Equal(jsonLiteral))
		})
	})
})

var _ = Describe("SetKey", func() {
	Context("with valid key and value", func() {
		DescribeTable("should set the log-level to the provided value", func(value string) {
			err := SetKey("log-level", value)
			Expect(err).ToNot(HaveOccurred())
			contentBytes, err := os.ReadFile(configFilePath)
			Expect(err).ToNot(HaveOccurred())
			content := string(contentBytes)
			Expect(strings.Contains(content, value)).To(BeTrue())
		},
			Entry("log-level info", "info"),
			Entry("log-level debug", "debug"),
			Entry("log-level error", "error"),
		)

		DescribeTable("should set the log-format to the provided value", func(value string) {
			err := SetKey("log-format", value)
			Expect(err).ToNot(HaveOccurred())
			content := readConfigFileContent()
			Expect(strings.Contains(content, value)).To(BeTrue())
		},
			Entry("log-format pretty", "pretty"),
			Entry("log-format text", "text"),
			Entry("log-format json", jsonLiteral),
		)

		DescribeTable("should set the output to the provided value", func(value string) {
			err := SetKey("output", value)
			Expect(err).ToNot(HaveOccurred())
			content := readConfigFileContent()
			Expect(strings.Contains(content, value)).To(BeTrue())
		},
			Entry("output table", "pretty"),
			Entry("output yaml", "yaml"),
			Entry("output json", jsonLiteral),
		)

		It("should throw an error if the configuration file cannot be fetched", func() {
			errorMessage := "error fetching config file"
			configFileFuncs = configFileFunctions{
				getPath: func() (string, error) {
					return "", fmt.Errorf("%s", errorMessage)
				},
			}
			err := SetKey("log-level", "info")
			Expect(err).To(HaveOccurred())
			Expect(strings.Contains(err.Error(), errorMessage)).To(BeTrue())
		})

		It("should throw an error if the configuration file cannot be updated", func() {
			errorMessage := "error updating config file"
			configFileFuncs = configFileFunctions{
				updateFile: func(string, []byte) error {
					return fmt.Errorf("%s", errorMessage)
				},
				getPath: func() (string, error) {
					return configFilePath, nil
				},
			}
			err := SetKey("log-level", "info")
			Expect(err).To(HaveOccurred())
			Expect(strings.Contains(err.Error(), errorMessage)).To(BeTrue())
		})
	})

	Context("with invalid environment variable key", func() {
		It("should return an error", func() {
			err := SetKey("invalid-key", "value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'invalid-key' is not a valid configuration key"))
		})
	})

	Context("with invalid environment variable value", func() {
		It("should return an error", func() {
			err := SetKey("log-level", "value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("value 'value' is not valid for configuration key 'log-level'"))
			Expect(err.Error()).To(ContainSubstring("Supported values are: info, debug, error"))
		})

		It("should return an error", func() {
			err := SetKey("output", "value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("value 'value' is not valid for configuration key 'output'"))
			Expect(err.Error()).To(ContainSubstring("Supported values are: json, yaml, pretty"))
		})

		DescribeTable("should reject an invalid api-endpoint value",
			func(addr, fragment string) {
				err := SetKey("api-endpoint", addr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fragment))
			},
			Entry("empty string", "", "must not be empty"),
			Entry("no scheme", "localhost:8090", "scheme must be http or https"),
			Entry("wrong scheme", "ftp://host", "scheme must be http or https"),
			Entry("no host", "http://", "host must not be empty"),
		)

		It("should accept a valid api-endpoint value", func() {
			err := SetKey("api-endpoint", "https://api.example.com:8090")
			Expect(err).ToNot(HaveOccurred())
			Expect(readConfigFileContent()).To(ContainSubstring("https://api.example.com:8090"))
		})
	})

})

var _ = Describe("UnSetKey", func() {
	Context("with valid key", func() {
		It("should remove the log-level key from the configuration file", func() {
			setupConfigurationFile()

			err := UnSetKey("log-level")
			Expect(err).ToNot(HaveOccurred())

			content := readConfigFileContent()
			Expect(strings.Contains(content, "log-level")).ToNot(BeTrue())
		})

		It("should remove the log-format key from the configuration file", func() {
			setupConfigurationFile()

			err := UnSetKey("log-format")
			Expect(err).ToNot(HaveOccurred())

			content := readConfigFileContent()
			Expect(strings.Contains(content, "log-format")).ToNot(BeTrue())
		})

		It("should remove the output key from the configuration file", func() {
			setupConfigurationFile()

			err := UnSetKey("output")
			Expect(err).ToNot(HaveOccurred())

			content := readConfigFileContent()
			Expect(strings.Contains(content, "output")).ToNot(BeTrue())
		})

		It("should throw an error if the configuration file cannot be fetched", func() {
			errorMessage := "error reading config file"
			configFileFuncs = configFileFunctions{
				getPath: func() (string, error) {
					return "", fmt.Errorf("%s", errorMessage)
				},
			}
			err := SetKey("log-level", "info")
			Expect(err).To(HaveOccurred())
			Expect(strings.Contains(err.Error(), errorMessage)).To(BeTrue())
		})

		It("should throw an error if the configuration file cannot be updated", func() {
			errorMessage := "error updating config file"
			configFileFuncs = configFileFunctions{
				getPath: func() (string, error) {
					return configFilePath, nil
				},
				updateFile: func(string, []byte) error {
					return fmt.Errorf("%s", errorMessage)
				},
			}
			err := UnSetKey("log-level")
			Expect(err).To(HaveOccurred())
			Expect(strings.Contains(err.Error(), errorMessage)).To(BeTrue())
		})

	})

	Context("with invalid key", func() {
		It("should throw an error if the key does not exist", func() {
			setupConfigurationFile()

			err := UnSetKey("invalid-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'invalid-key' is not a valid configuration key"))
		})
	})
})

var _ = Describe("validateConfiguration", func() {
	Context("with valid configuration", func() {
		It("should validate the configuration struct", func() {
			Config.LogLevel = fileLogLevel
			Config.LogFormat = jsonLiteral
			Config.Output = jsonLiteral
			Config.APIEndpoint = "http://localhost:8090"

			err := validateConfig(&Config)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with invalid configuration", func() {
		It("should throw an error when the configuration is invalid", func() {
			Config.LogLevel = ""
			Config.LogFormat = jsonLiteral
			Config.Output = jsonLiteral
			Config.APIEndpoint = "http://localhost:8090"

			err := validateConfig(&Config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid log-level: "))
		})

		DescribeTable("should reject an invalid api-endpoint",
			func(addr, expectedErrFragment string) {
				Config.LogLevel = fileLogLevel
				Config.LogFormat = jsonLiteral
				Config.Output = jsonLiteral
				Config.APIEndpoint = addr

				err := validateConfig(&Config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedErrFragment))
			},
			Entry("empty", "", "must not be empty"),
			Entry("no scheme", "localhost:8090", "scheme must be http or https"),
			Entry("wrong scheme", "ftp://localhost:8090", "scheme must be http or https"),
			Entry("no host", "http://", "host must not be empty"),
		)

		DescribeTable("should accept a valid api-endpoint",
			func(addr string) {
				Config.LogLevel = fileLogLevel
				Config.LogFormat = jsonLiteral
				Config.Output = jsonLiteral
				Config.APIEndpoint = addr

				Expect(validateConfig(&Config)).To(Succeed())
			},
			Entry("http with port", "http://localhost:8090"),
			Entry("https with port", "https://api.konfidence.example.com:8090"),
			Entry("http no port", "http://konfidence-api.konfidence-system"),
			Entry("https no port", "https://api.example.com"),
		)
	})
})

func setupConfigurationFile() {
	content := []byte(`{"log-level": "info", "log-format": "json", "output": "json", "api-endpoint": "http://localhost:8090"}`)
	err := os.WriteFile(configFilePath, content, 0644)
	Expect(err).ToNot(HaveOccurred())
}

func readConfigFileContent() string {
	contentBytes, err := os.ReadFile(configFilePath)
	Expect(err).ToNot(HaveOccurred())
	content := string(contentBytes)
	return content
}
