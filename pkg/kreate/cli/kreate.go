package cli

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/kreate/cli/project"
	"github.com/spf13/cobra"
)

const commandName = "kreate"

func NewCmdKreate() *cobra.Command {
	kreate := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Easily create Kubernetes applications",
		Long:  `Easily create and manage Kubernetes applications using the Component operator created by the Snowdrop team.`,
	}

	kreate.AddCommand(project.NewCmdProject(commandName))

	return kreate
}