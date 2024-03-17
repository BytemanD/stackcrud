package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BytemanD/easygo/pkg/global/gitutils"
	"github.com/BytemanD/easygo/pkg/global/logging"

	"github.com/BytemanD/skyman/cli/compute"
	"github.com/BytemanD/skyman/cli/identity"
	"github.com/BytemanD/skyman/cli/image"
	"github.com/BytemanD/skyman/cli/networking"
	"github.com/BytemanD/skyman/cli/prune"
	"github.com/BytemanD/skyman/cli/storage"
	"github.com/BytemanD/skyman/cli/templates"
	"github.com/BytemanD/skyman/common"
	"github.com/BytemanD/skyman/common/i18n"
	"github.com/BytemanD/skyman/openstack"
	"github.com/BytemanD/skyman/utility"
)

var (
	Version       string
	GoVersion     string
	BuildDate     string
	BuildPlatform string
)

var LOGO = `
      _                             
  ___| |  _ _   _ ____  _____ ____  
 /___) |_/ ) | | |    \(____ |  _ \ 
|___ |  _ (| |_| | | | / ___ | | | |
(___/|_| \_)\__  |_|_|_\_____|_| |_|
           (____/
`

func getVersion() string {
	if Version == "" {
		return gitutils.GetVersion()
	}
	return fmt.Sprint(Version)
}

var pruneCmd = &cobra.Command{Use: "prune", Short: "prune resources"}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version of client and server",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, _ []string) {
		if GoVersion == "" {
			GoVersion = runtime.Version()
		}
		fmt.Println("Client:")
		fmt.Printf("  %-14s: %s\n", "Version", getVersion())
		fmt.Printf("  %-14s: %s\n", "GoVersion", GoVersion)
		fmt.Printf("  %-14s: %s\n", "BuildDate", BuildDate)
		fmt.Printf("  %-14s: %s\n", "BuildPlatform", BuildPlatform)

		client := openstack.DefaultClient()

		fmt.Println("Servers:")

		identityVerion, err := client.KeystoneV3().GetStableVersion()
		utility.LogError(err, "get idendity veresion failed", true)
		if err != nil {
			return
		}
		fmt.Printf("  %-11s: %s\n", "Identity", identityVerion.VersoinInfo())

		errors := 0
		computeVerion, err := client.NovaV2().GetCurrentVersion()
		if err == nil {
			fmt.Printf("  %-11s: %s\n", "Compute", computeVerion.VersoinInfo())
		} else {
			fmt.Printf("  %-11s: Unknown (%s)\n", "Compute", err)
			errors++
		}
		imageVerion, err := client.GlanceV2().GetCurrentVersion()
		if err == nil {
			fmt.Printf("  %-11s: %s\n", "Image", imageVerion.VersoinInfo())
		} else {
			fmt.Printf("  %-11s: Unknown (%s)\n", "Image", err)
			errors++
		}
		storageVerion, err := client.CinderV2().GetCurrentVersion()
		if err == nil {
			fmt.Printf("  %-11s: %s\n", "Storage", storageVerion.VersoinInfo())
		} else {
			fmt.Printf("  %-11s: Unknown (%s)\n", "Storage", err)
			errors++
		}
		networkingVerion, err := client.NeutronV2().GetCurrentVersion()
		if err == nil {
			fmt.Printf("  %-11s: %s\n", "Networking", networkingVerion.VersoinInfo())
		} else {
			fmt.Printf("  %-11s: Unknown (%s)\n", "Networking", err)
			errors++
		}
		if errors > 0 {
			os.Exit(1)
		}
	},
}

func main() {
	rootCmd := cobra.Command{
		Use:     "skyman",
		Short:   "Golang OpenStack Client \n" + LOGO,
		Version: getVersion(),
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			format, _ := cmd.Flags().GetString("format")
			if format != "" && !utility.StringsContain(common.GetOutputFormats(), format) {
				fmt.Printf("invalid foramt '%s'\n", format)
				os.Exit(1)
			}
			conf, _ := cmd.Flags().GetString("conf")
			if err := common.LoadConfig(conf); err != nil {
				fmt.Printf("load config failed, %v\n", err)
				os.Exit(1)
			}
			if common.CONF.Debug {
				logging.BasicConfig(logging.LogConfig{Level: logging.DEBUG})
			} else {
				logging.BasicConfig(logging.LogConfig{Level: logging.INFO})
			}
			logging.Debug("load config file from %s", viper.ConfigFileUsed())
		},
	}

	rootCmd.PersistentFlags().BoolP("debug", "d", false, i18n.T("showDebug"))
	rootCmd.PersistentFlags().StringP("conf", "c", os.Getenv("SKYMAN_CONF_FILE"),
		i18n.T("thePathOfConfigFile"))
	rootCmd.PersistentFlags().StringP("format", "f", "table",
		fmt.Sprintf(i18n.T("formatAndSupported"), common.GetOutputFormats()))

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))

	pruneCmd.AddCommand(
		prune.PortPrune,
		prune.ServerPrune,
		prune.VolumePrune,
	)

	rootCmd.AddCommand(
		versionCmd,
		identity.Token,
		identity.Service, identity.Endpoint, identity.Region,
		identity.User, identity.Project,
		compute.Server, compute.Flavor, compute.Hypervisor,
		compute.Keypair, compute.Compute, compute.Console,
		compute.Migration, compute.AZ, compute.Aggregate,
		image.Image,
		storage.Volume,
		networking.Router, networking.Network, networking.Subnet, networking.Port,
		templates.CreateCmd, templates.DeleteCmd,
		pruneCmd,
	)
	rootCmd.Execute()
}
