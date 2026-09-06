package cli

import (
	"fmt"

	"github.com/GunithaR/ShipHold/internal/application/check"
	"github.com/GunithaR/ShipHold/internal/config"
	"github.com/GunithaR/ShipHold/internal/infrastructure/git"
	"github.com/spf13/cobra"
)

func newCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check deployment readiness",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load("examples/policy.yaml")
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			provider := git.Provider{}
			service := check.NewService(provider)

			result := service.Run(cfg.Policy)

			fmt.Println("Decision:", result.Decision)

			for _, reason := range result.Reasons {
				fmt.Println("-", reason)
			}
		},
	}
}
