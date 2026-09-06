package cli

import (
	"fmt"

	"github.com/GunithaR/ShipHold/internal/application/check"
	"github.com/GunithaR/ShipHold/internal/infrastructure/git"
	"github.com/spf13/cobra"
)

func NewCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check depoyment readiness",
		Run: func(cmd *cobra.Command, args []string) {
			provider := git.Provider{}
			service := check.NewService(provider)

			service.Run()

			fmt.Println("Check completed")
		},
	}
}
