package cli

import (
	"fmt"

	"github.com/oaago/cli/utils"
	"github.com/spf13/cobra"
)

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "update oaacli version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("将会自动更新到master分支 请稍等....")
		utils.RunCmd("go install github.com/oaago/oaacli@master", true)
		utils.RunCmd("go install github.com/oaago/protoc-gen-oaago@main", true)
		fmt.Println("更新完成")
	},
}
