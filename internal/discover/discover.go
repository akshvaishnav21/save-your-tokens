package discover

import (
	"time"

	"github.com/saveyourtokens/syt/internal/registry"
)

// CommandCount holds a command string and its occurrence count.
type CommandCount struct {
	Command  string
	Count    int
	SytCmd   string
	SavesPct int
	Category registry.Category
}

// DiscoverResult holds the result of analyzing Claude Code sessions.
type DiscoverResult struct {
	Supported   []CommandCount
	Unsupported []CommandCount
	AlreadySyt  []CommandCount
	TotalCmds   int
	Since       time.Time
	FilesScanned int
}

// Options controls the discover behavior.
type Options struct {
	ProjectPath string
	Since       time.Time
	AllProjects bool
}

// Analyze scans session files and classifies all Bash commands.
func Analyze(provider SessionProvider, opts Options) (*DiscoverResult, error) {
	files, err := provider.Sessions(opts.ProjectPath, opts.Since, opts.AllProjects)
	if err != nil {
		return nil, err
	}

	result := &DiscoverResult{
		Since:       opts.Since,
		FilesScanned: len(files),
	}

	// Aggregate counts
	supported := make(map[string]*CommandCount)
	unsupported := make(map[string]*CommandCount)
	alreadySyt := make(map[string]*CommandCount)

	for _, f := range files {
		cmds, err := ExtractBashCommands(f)
		if err != nil {
			continue
		}
		for _, cmd := range cmds {
			result.TotalCmds++
			c := registry.ClassifyCommand(cmd)
			switch c.Kind {
			case "supported":
				key := c.SytCmd
				if key == "" {
					key = cmd
				}
				if existing, ok := supported[key]; ok {
					existing.Count++
				} else {
					supported[key] = &CommandCount{
						Command:  cmd,
						Count:    1,
						SytCmd:   c.SytCmd,
						SavesPct: c.SavesPct,
						Category: c.Category,
					}
				}
			case "ignored":
				if c.SytCmd != "" {
					// Already using syt
					if existing, ok := alreadySyt[cmd]; ok {
						existing.Count++
					} else {
						alreadySyt[cmd] = &CommandCount{Command: cmd, Count: 1}
					}
				}
			default: // unsupported
				if existing, ok := unsupported[cmd]; ok {
					existing.Count++
				} else {
					unsupported[cmd] = &CommandCount{Command: cmd, Count: 1}
				}
			}
		}
	}

	// Convert maps to slices (sorted by count desc)
	for _, v := range supported {
		result.Supported = append(result.Supported, *v)
	}
	for _, v := range unsupported {
		result.Unsupported = append(result.Unsupported, *v)
	}
	for _, v := range alreadySyt {
		result.AlreadySyt = append(result.AlreadySyt, *v)
	}

	// Sort by count descending
	sortByCount(result.Supported)
	sortByCount(result.Unsupported)
	sortByCount(result.AlreadySyt)

	return result, nil
}

func sortByCount(s []CommandCount) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Count > s[j-1].Count; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
