package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"github.com/Cloud-Pie/SPDT/util"
	db "github.com/Cloud-Pie/SPDT/storage"
)

// policiesCmd represents the delete policies command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove policy",
	Long: "Remove a stored policy",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("called delete")
		id := ""
		policyDAO := db.PolicyDAO{
			Server:util.DEFAULT_DB_SERVER_POLICIES,
			Database:util.DEFAULT_DB_POLICIES,
		}
		policyDAO.Connect()
		err := policyDAO.DeleteById(id)

		if err != nil {
			fmt.Println("Error, policy could not be deleted")
			fmt.Errorf(err.Error())
		}
	},
}
