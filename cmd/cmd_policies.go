package cmd

import (
	db "github.com/Cloud-Pie/SPDT/storage"
	"encoding/json"
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"time"
)

// policiesCmd represents the policies command
var policiesCmd = &cobra.Command {
	Use:   "policies",
	Short: "List policies",
	Long: `List stored policies.`,
	Run: retrieve,
}

var all bool

func retrieve (cmd *cobra.Command, args []string) {
	id := cmd.Flag("pId").Value.String()
	start := cmd.Flag("start-time").Value.String()
	end := cmd.Flag("end-time").Value.String()

	policyDAO := db.GetPolicyDAO()
	policyDAO.Connect()

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
		layout := "2014-09-12 11:45:26"
		startTime,err := time.Parse(layout , start)
		check(err)
		endTime,err := time.Parse(layout , end)
		check(err)
		policies,err := policyDAO.FindOneByTimeWindow(startTime, endTime)
		check(err)
		writeToFile(policies)
	}else if all {
		policies,err := policyDAO.FindAll()
		check(err)
		writeToFile(policies)
	}
}

func init() {
	policiesCmd.Flags().String("start-time", "", "Start time of the horizon span")
	policiesCmd.Flags().String("end-time", "", "End time of the horizon span")
	policiesCmd.Flags().String("pId", "", "Policy ID")
	policiesCmd.Flags().BoolVar(&all,"all", false, "Retrieve all stored policies")
}

func check(e error) {
	if e != nil {
		fmt.Println("An error has occurred")
		panic(e)
	}
}

func writeToFile(out interface{}) {
	data, err := json.Marshal(out)
	check(err)
	output := string(data)
	f, err := os.Create("output.json")
	check(err)
	_,err = f.WriteString(output)
	check(err)
	fmt.Println("See the output of your query in the file output.json")
}