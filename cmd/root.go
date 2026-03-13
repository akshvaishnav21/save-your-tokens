package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var verboseCount int

var rootCmd = &cobra.Command{
	Use:   "syt",
	Short: "Token-optimized CLI proxy for AI coding assistants",
	Long: `SaveYourTokens (syt) is a transparent CLI proxy for AI coding assistants.
It intercepts terminal commands, rewrites them to token-optimized equivalents,
compresses verbose output, and tracks cumulative token savings.`,
}

// Execute runs the root command and returns any error.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().CountVarP(&verboseCount, "verbose", "v", "Increase verbosity (can be repeated)")
}
