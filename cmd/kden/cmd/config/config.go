package config

import (
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
)

var commonDescription = `
The CLI uses a configuration file to store settings and preferences. 

The file is stored in the $XDG_CONFIG_HOME/kden/config.json directory. 
The value of $XDG_CONFIG_HOME depends on the operating system. 
For more information, check the XDG Base Directory documentation: https://specifications.freedesktop.org/basedir/latest/.` //nolint:lll

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the CLI's configuration",
	Long:  commonDescription,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

var setConfigCmd = &cobra.Command{
	Use:   "set <configuration_property> <value>",
	Short: "Set a value for a configuration property.",
	Long: fmt.Sprintf(
		`This command sets a value for a configuration property inside the CLI's configuration file.
The accepted values are:
%s
Additional information: 
%s`, getAcceptedConfigurationKeyValuePairs(), commonDescription),
	Args: cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			keys := make([]string, 0, len(cfg.SupportedConfigurations))
			for key := range cfg.SupportedConfigurations {
				keys = append(keys, key)
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		case 1:
			return cfg.SupportedConfigurations[args[0]], cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		log.Infof("Setting configuration: %s = %s", key, value)
		err := cfg.SetKey(key, value)
		if err != nil {
			cmd.PrintErrln("Error while setting configuration")
			return fmt.Errorf("failed to set configuration: %w", err)
		}
		return nil
	},
}

var unsetConfigCmd = &cobra.Command{
	Use:   "unset <configuration_property>",
	Short: "Unset a configuration property.",
	Long: fmt.Sprintf(
		`This command unsets a configuration property inside the CLI's configuration file.
The accepted values are: %s

Additional information:
%s`, getAcceptedConfigurationKeysAsArray(), commonDescription),
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getAcceptedConfigurationKeysAsArray(), cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		log.Infof("Unsetting configuration '%s' ", key)
		err := cfg.UnSetKey(key)
		if err != nil {
			cmd.PrintErrln("Error while unsetting configuration")
			return fmt.Errorf("failed to unset configuration %q: %w", key, err)

		}
		return nil
	},
}

func NewConfigCmd() *cobra.Command {
	configCmd.AddCommand(setConfigCmd)
	configCmd.AddCommand(unsetConfigCmd)
	return configCmd
}

func getAcceptedConfigurationKeyValuePairs() string {
	var result string

	for key, value := range cfg.SupportedConfigurations {
		result = result + fmt.Sprintf("'%s' - %s\n", key, value)
	}
	return result
}

func getAcceptedConfigurationKeysAsArray() []string {
	result := make([]string, 0, len(cfg.SupportedConfigurations))
	for key := range cfg.SupportedConfigurations {
		result = append(result, key)
	}
	return result
}
