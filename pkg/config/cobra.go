package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// CobraPreRunHook can be used with cobra CLI library PreRunE feature to initialize
// required  configurations before  running commands. Config struct  pointed by the
// passed in pointer will be populated with the configs when the hook is executed.
func CobraPreRunHook(envPrefix, configName string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		configFile, _ := cmd.Flags().GetString("config")
		if err := viperInit(envPrefix, configName, configFile); err != nil {
			return err
		}
		bindFlags(envPrefix, cmd, viper.GetViper())
		return nil
	}
}

func bindFlags(envPrefix string, cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			_ = v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		}

		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
