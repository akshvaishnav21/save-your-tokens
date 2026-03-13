package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var (
	gainHistory    bool
	gainDaily      bool
	gainGraph      bool
	gainSinceDays  int
	gainFormat     string
)

var gainCmd = &cobra.Command{
	Use:   "gain",
	Short: "Show token savings dashboard",
	RunE:  runGain,
}

func runGain(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	dbPath := cfg.Tracking.DatabasePath
	if dbPath == "" {
		dbPath = utils.DataDir() + "/syt.db"
	}

	t, err := tracker.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("opening tracker: %w", err)
	}
	defer t.Close()

	since := time.Now().AddDate(0, 0, -gainSinceDays)
	summary, err := t.GetSummary(since)
	if err != nil {
		return fmt.Errorf("getting summary: %w", err)
	}

	if gainFormat == "json" {
		b, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if gainHistory {
		records, err := t.GetHistory(50)
		if err != nil {
			return fmt.Errorf("getting history: %w", err)
		}
		fmt.Printf("Recent commands (last %d):\n", len(records))
		fmt.Printf("%-30s  %-8s  %-8s  %s\n", "Command", "In", "Out", "Savings")
		fmt.Println(strings.Repeat("─", 70))
		for _, r := range records {
			savings := 0.0
			if r.InputTokens > 0 {
				savings = 100.0 - float64(r.OutputTokens)/float64(r.InputTokens)*100.0
			}
			fmt.Printf("%-30s  %-8s  %-8s  %s\n",
				utils.Truncate(r.SytCmd, 30),
				utils.FormatTokens(r.InputTokens),
				utils.FormatTokens(r.OutputTokens),
				utils.FormatSavingsPct(savings),
			)
		}
		return nil
	}

	if gainDaily {
		stats, err := t.GetDailyStats(gainSinceDays)
		if err != nil {
			return fmt.Errorf("getting daily stats: %w", err)
		}
		fmt.Printf("Daily breakdown (last %d days):\n", gainSinceDays)
		fmt.Printf("%-12s  %-10s  %s\n", "Date", "Commands", "Tokens Saved")
		fmt.Println(strings.Repeat("─", 40))
		for _, s := range stats {
			fmt.Printf("%-12s  %-10d  %s\n", s.Date, s.Commands, utils.FormatTokens(s.TokensSaved))
		}
		return nil
	}

	if gainGraph {
		printGainGraph(summary)
		return nil
	}

	// Default dashboard
	fmt.Println("SaveYourTokens — Token Savings Report")
	fmt.Println(strings.Repeat("━", 38))
	fmt.Printf("Period:      Last %d days\n", gainSinceDays)
	fmt.Printf("Commands:    %s tracked\n", formatInt(summary.TotalCommands))
	fmt.Printf("Saved:       %s tokens\n", formatInt(summary.TotalSaved))
	fmt.Printf("Avg savings: %s\n", utils.FormatSavingsPct(summary.AvgSavingsPct))
	fmt.Println()

	if len(summary.ByCommand) > 0 {
		fmt.Println("Top commands by tokens saved:")
		for i, cs := range summary.ByCommand {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-20s │ %5d runs │ %8s tokens │ %s\n",
				cs.Command,
				cs.Runs,
				utils.FormatTokens(cs.TokensSaved),
				utils.FormatSavingsPct(cs.AvgSavingsPct),
			)
		}
	}

	return nil
}

func printGainGraph(summary tracker.Summary) {
	fmt.Printf("Token savings — last %d days\n", gainSinceDays)
	fmt.Println(strings.Repeat("─", 50))

	if len(summary.ByDay) == 0 {
		fmt.Println("No data")
		return
	}

	maxSaved := 0
	for _, d := range summary.ByDay {
		if d.TokensSaved > maxSaved {
			maxSaved = d.TokensSaved
		}
	}
	if maxSaved == 0 {
		fmt.Println("No savings yet")
		return
	}

	barWidth := 30
	// Show in chronological order (reverse of ByDay which is DESC)
	days := summary.ByDay
	for i := len(days) - 1; i >= 0; i-- {
		d := days[i]
		width := d.TokensSaved * barWidth / maxSaved
		bar := strings.Repeat("█", width)
		fmt.Printf("%-12s │ %-30s %s\n", d.Date, bar, utils.FormatTokens(d.TokensSaved))
	}
}

func formatInt(n int) string {
	// Simple comma formatting
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func init() {
	gainCmd.Flags().BoolVar(&gainHistory, "history", false, "Show recent command log")
	gainCmd.Flags().BoolVar(&gainDaily, "daily", false, "Show day-by-day breakdown")
	gainCmd.Flags().BoolVar(&gainGraph, "graph", false, "Show ASCII bar chart")
	gainCmd.Flags().IntVar(&gainSinceDays, "since", 30, "Days to look back")
	gainCmd.Flags().StringVar(&gainFormat, "format", "", "Output format (json)")
	rootCmd.AddCommand(gainCmd)
}
