package cmd

import (
	db "github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/server"
	"encoding/json"
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"time"
	"github.com/Cloud-Pie/SPDT/util"
)

// policiesCmd represents the policies command
var policiesCmd = &cobra.Command {
	Use:   "policies",
	Short: "List policies",
	Long: `List stored policies.`,
	Run: retrieve,
}

var all bool

func init() {
	policiesCmd.Flags().String("start-time", "", "Start time of the horizon span")
	policiesCmd.Flags().String("end-time", "", "End time of the horizon span")
	policiesCmd.Flags().String("pId", "", "Policy ID")
	policiesCmd.Flags().BoolVar(&all,"all", false, "Retrieve all stored policies")
	policiesCmd.Flags().String("config-file", "config.yml", "Configuration file path")
}

func retrieve (cmd *cobra.Command, args []string) {
	id := cmd.Flag("pId").Value.String()
	start := cmd.Flag("start-time").Value.String()
	end := cmd.Flag("end-time").Value.String()
	configFile := cmd.Flag("config-file").Value.String()
	systemConfiguration := server.ReadSysConfigurationFile(configFile)
	policyDAO := db.GetPolicyDAO(systemConfiguration.MainServiceName)

	if id != "" {
		policy,err := policyDAO.FindByID(id)
		if err != nil {
			fmt.Println("Error, policy could not be deleted")
			fmt.Println(err.Error())
		} else {
			fmt.Println("Policy retrieved")
			writeToFile(policy)
		}
	} else if start != "" && end != "" {
		startTime,err := time.Parse(util.UTC_TIME_LAYOUT , start)
		check(err, "Invalid start time")
		endTime,err := time.Parse(util.UTC_TIME_LAYOUT , end)
		check(err, "Invalid end time")
		policies,err := policyDAO.FindAllByTimeWindow(startTime, endTime)
		check(err, "No policies for the specified window")
		if len(policies) == 0 {
			log.Fatalf("No policies found for the specified window")
			fmt.Println("No policies found for the specified window")
		}
		writeToFile(policies)
	}else if all {
		policies,err := policyDAO.FindAll()
		check(err, "No policies for the specified window")
		writeToFile(policies)
	}
}

func writeToFile(out interface{}) {
	data, err := json.Marshal(out)
	check(err, "Error writing File")
	output := string(data)
	f, err := os.Create("output.json")
	check(err, "Error writing File")
	_,err = f.WriteString(output)
	check(err, "Error writing File")
	fmt.Println("See the output of your query in the file output.json")
}