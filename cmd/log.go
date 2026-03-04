package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/justEstif/beans/internal/bean"
	"github.com/justEstif/beans/internal/beancore"
	"github.com/spf13/cobra"
)

var (
	logFormat     string
	logLimit      int
	logSince      string
	logShowBody   bool
	logShowEvents bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent bean changes",
	Long: `Displays a log of recent changes to beans, including creation, updates, and deletions.
This provides a historical view of bean activity.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get all beans with their modification times
		beans := core.All()
		
		// Create change entries
		var entries []changeEntry
		
		for _, b := range beans {
			// Get file info for modification time
			filePath := core.FullPath(b)
			info, err := os.Stat(filePath)
			if err != nil {
				continue
			}
			
			entry := changeEntry{
				ID:        b.ID,
				Type:      beancore.EventUpdated,
				Timestamp: info.ModTime(),
				Bean:      b,
			}
			
			// If no created time, assume it's the same as mod time
			if b.CreatedAt == nil {
				entry.Type = beancore.EventCreated
			} else if info.ModTime().Sub(*b.CreatedAt) < time.Second {
				entry.Type = beancore.EventCreated
			}
			
			entries = append(entries, entry)
		}
		
		// Sort by timestamp (newest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		})
		
		// Apply limit
		if logLimit > 0 && len(entries) > logLimit {
			entries = entries[:logLimit]
		}
		
		// Filter by since time
		if logSince != "" {
			duration, err := time.ParseDuration(logSince)
			if err != nil {
				return fmt.Errorf("invalid duration format: %s (use formats like '1h', '30m', '7d')", logSince)
			}
			
			sinceTime := time.Now().Add(-duration)
			var filtered []changeEntry
			for _, entry := range entries {
				if entry.Timestamp.After(sinceTime) {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}
		
		// Print entries
		for _, entry := range entries {
			printChangeEntry(entry)
		}
		
		return nil
	},
}

type changeEntry struct {
	ID        string
	Type      beancore.EventType
	Timestamp time.Time
	Bean      *bean.Bean
}

func init() {
	logCmd.Flags().StringVarP(&logFormat, "format", "f", "simple", "Output format: simple, detailed, json")
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Maximum number of entries to show")
	logCmd.Flags().StringVar(&logSince, "since", "", "Show changes since this duration (e.g., '1h', '30m', '7d')")
	logCmd.Flags().BoolVar(&logShowBody, "body", false, "Include bean body in output")
	logCmd.Flags().BoolVarP(&logShowEvents, "events", "e", false, "Show event type (created/updated)")
	
	rootCmd.AddCommand(logCmd)
}

func printChangeEntry(entry changeEntry) {
	switch logFormat {
	case "json":
		printJSONEntry(entry)
	case "detailed":
		printDetailedEntry(entry)
	default: // simple
		printSimpleEntry(entry)
	}
}

func printSimpleEntry(entry changeEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	var eventIcon string
	
	if logShowEvents {
		switch entry.Type {
		case beancore.EventCreated:
			eventIcon = "🆕 "
		case beancore.EventUpdated:
			eventIcon = "✏️ "
		default:
			eventIcon = "📝 "
		}
	}
	
	fmt.Printf("%s %s%s", timestamp, eventIcon, entry.ID)
	
	if entry.Bean.Title != "" {
		fmt.Printf(" - %s", entry.Bean.Title)
	}
	
	fmt.Println()
}

func printDetailedEntry(entry changeEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	
	var action string
	var emoji string
	switch entry.Type {
	case beancore.EventCreated:
		action = "Created"
		emoji = "🆕"
	case beancore.EventUpdated:
		action = "Updated"
		emoji = "✏️ "
	default:
		action = "Modified"
		emoji = "📝"
	}
	
	fmt.Printf("%s %s %s: %s\n", timestamp, emoji, action, entry.ID)
	fmt.Printf("   Title: %s\n", entry.Bean.Title)
	
	if len(entry.Bean.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(entry.Bean.Tags, ", "))
	}
	
	if entry.Bean.Priority != "" {
		fmt.Printf("   Priority: %s\n", entry.Bean.Priority)
	}
	
	if entry.Bean.Status != "" {
		fmt.Printf("   Status: %s\n", entry.Bean.Status)
	}
	
	if logShowBody && entry.Bean.Body != "" {
		fmt.Printf("   Body:\n")
		bodyLines := strings.Split(entry.Bean.Body, "\n")
		for _, line := range bodyLines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("     %s\n", line)
			}
		}
	}
	
	fmt.Println()
}

func printJSONEntry(entry changeEntry) {
	data := map[string]interface{}{
		"timestamp": entry.Timestamp.Format(time.RFC3339),
		"id":        entry.ID,
		"type":      entry.Type.String(),
		"bean":      entry.Bean,
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("{\"error\": \"failed to marshal JSON: %v\"}\n", err)
		return
	}
	fmt.Println(string(jsonData))
}