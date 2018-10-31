package cmd

import (
	"github.com/spf13/cobra"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/server"
	"github.com/Cloud-Pie/SPDT/storage"
)

// policiesCmd represents the delete policies command
var updateProfilesCmd = &cobra.Command{
	Use:   "new-profiles",
	Short: "Update stored application profiles",
	Long: "Update stored application profiles",
	Run: updateProfiles,
}

func init() {
	updateProfilesCmd.Flags().String("config-file", "config.yml", "Configuration file path")
}

func updateProfiles(cmd *cobra.Command, args []string) {

	configFile := cmd.Flag("config-file").Value.String()
	systemConfiguration,_ := util.ReadConfigFile(configFile)
	profilesDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)
	err := profilesDAO.DeleteAll()
	check(err, "Error removing old profiles.")

	vmBootingProfileDAO := storage.GetVMBootingProfileDAO()
	err = vmBootingProfileDAO.DeleteAll()
	check(err, "Error removing old profiles.")

	err = server.FetchApplicationProfile(systemConfiguration)
	check(err, "No application profiles found.")
	vmProfiles,err2 := server.ReadVMProfiles()
	check(err2, "No VM profiles found.")
	err2 = server.FetchVMBootingProfiles(systemConfiguration,vmProfiles)
	check(err2, "No VM booting times found.")
}
