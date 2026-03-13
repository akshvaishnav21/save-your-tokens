package discover

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatText formats a DiscoverResult as human-readable text.
func FormatText(r *DiscoverResult) string {
	var sb strings.Builder

	sb.WriteString("SaveYourTokens — Session Discovery Report\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(&sb, "Files scanned:  %d\n", r.FilesScanned)
	fmt.Fprintf(&sb, "Total commands: %d\n\n", r.TotalCmds)

	if len(r.Supported) > 0 {
		sb.WriteString("Commands that could use syt (token savings available):\n")
		for _, c := range r.Supported {
			if len(c.Command) > 50 {
				c.Command = c.Command[:47] + "..."
			}
			fmt.Fprintf(&sb, "  %-50s  %4d×  → %s  (%d%% savings)\n",
				c.Command, c.Count, c.SytCmd, c.SavesPct)
		}
		sb.WriteString("\n")
	}

	if len(r.AlreadySyt) > 0 {
		sb.WriteString("Already using syt:\n")
		for _, c := range r.AlreadySyt {
			fmt.Fprintf(&sb, "  %-50s  %4d×\n", c.Command, c.Count)
		}
		sb.WriteString("\n")
	}

	if len(r.Unsupported) > 0 && len(r.Unsupported) <= 10 {
		sb.WriteString("Other commands (not yet supported):\n")
		for _, c := range r.Unsupported {
			if len(c.Command) > 50 {
				c.Command = c.Command[:47] + "..."
			}
			fmt.Fprintf(&sb, "  %-50s  %4d×\n", c.Command, c.Count)
		}
	}

	return sb.String()
}

// FormatJSON formats a DiscoverResult as JSON.
func FormatJSON(r *DiscoverResult) (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
