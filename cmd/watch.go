package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/justEstif/beans/internal/beancore"
	"github.com/spf13/cobra"
)

var (
	watchFormat    string
	watchFollow    bool
	watchSince     string
	watchQuiet     bool
	watchJSON      bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for bean changes in real-time",
	Long: `Monitors the .beans directory for file changes and displays them in real-time.
Useful for seeing when beans are created, updated, or deleted by other processes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Start file watching
		if err := core.StartWatching(); err != nil {
			return fmt.Errorf("starting watcher: %w", err)
		}
		defer core.Unwatch()

		// Subscribe to bean events
		eventCh, unsubscribe := core.Subscribe()
		defer unsubscribe()

		// Set up signal handling for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Print initial message if not quiet
		if !watchQuiet {
			fmt.Printf("👀 Watching for bean changes in %s...\n", core.Root())
			fmt.Println("Press Ctrl+C to stop")
			fmt.Println()
		}

		// Process events
		for {
			select {
			case events := <-eventCh:
				if len(events) == 0 {
					continue
				}

				for _, event := range events {
					if !watchQuiet {
						printBeanEvent(event)
					}
				}

			case <-sigCh:
				if !watchQuiet {
					fmt.Println("\n⏹️  Stopping watcher...")
				}
				return nil
			}
		}
	},
}

func init() {
	watchCmd.Flags().StringVarP(&watchFormat, "format", "f", "simple", "Output format: simple, detailed, json")
	watchCmd.Flags().BoolVarP(&watchFollow, "follow", "F", true, "Follow changes in real-time")
	watchCmd.Flags().StringVar(&watchSince, "since", "", "Show changes since this time (e.g., '1h', '30m')")
	watchCmd.Flags().BoolVarP(&watchQuiet, "quiet", "q", false, "Suppress output messages")
	watchCmd.Flags().BoolVar(&watchJSON, "json", false, "Output events as JSON")
	
	rootCmd.AddCommand(watchCmd)
}

func printBeanEvent(event beancore.BeanEvent) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	switch watchFormat {
	case "json":
		printJSONEvent(event, timestamp)
	case "detailed":
		printDetailedEvent(event, timestamp)
	default: // simple
		printSimpleEvent(event, timestamp)
	}
}

func printSimpleEvent(event beancore.BeanEvent, timestamp string) {
	var emoji string
	switch event.Type {
	case beancore.EventCreated:
		emoji = "🆕"
	case beancore.EventUpdated:
		emoji = "✏️ "
	case beancore.EventDeleted:
		emoji = "🗑️ "
	default:
		emoji = "❓"
	}

	fmt.Printf("%s %s %s\n", timestamp, emoji, event.BeanID)
}

func printDetailedEvent(event beancore.BeanEvent, timestamp string) {
	var action string
	var emoji string
	switch event.Type {
	case beancore.EventCreated:
		action = "Created"
		emoji = "🆕"
	case beancore.EventUpdated:
		action = "Updated"
		emoji = "✏️ "
	case beancore.EventDeleted:
		action = "Deleted"
		emoji = "🗑️ "
	default:
		action = "Unknown"
		emoji = "❓"
	}

	fmt.Printf("%s %s %s: %s\n", timestamp, emoji, action, event.BeanID)
	
	if event.Bean != nil && event.Type != beancore.EventDeleted {
		fmt.Printf("   Title: %s\n", event.Bean.Title)
		if len(event.Bean.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(event.Bean.Tags, ", "))
		}
		if event.Bean.Priority != "" {
			fmt.Printf("   Priority: %s\n", event.Bean.Priority)
		}
		fmt.Println()
	}
}

func printJSONEvent(event beancore.BeanEvent, timestamp string) {
	data := map[string]interface{}{
		"timestamp": timestamp,
		"type":      event.Type.String(),
		"bean_id":   event.BeanID,
	}

	if event.Bean != nil {
		data["bean"] = event.Bean
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("{\"error\": \"failed to marshal JSON: %v\"}\n", err)
		return
	}
	fmt.Println(string(jsonData))
}