package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Edcko/techne-code/internal/db"
)

func newSessionCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
		Long:  "Manage conversation sessions - list, delete, and inspect past sessions.",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all sessions",
		Long:    "List all saved conversation sessions with their IDs and metadata.",
		Aliases: []string{"ls"},
		RunE:    runSessionList,
	}
	cmd.AddCommand(listCmd)

	deleteCmd := &cobra.Command{
		Use:     "delete [session-id]",
		Short:   "Delete a session",
		Long:    "Delete a conversation session by its ID.",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE:    runSessionDelete,
	}
	cmd.AddCommand(deleteCmd)

	showCmd := &cobra.Command{
		Use:   "show [session-id]",
		Short: "Show session details",
		Long:  "Display detailed information about a specific session.",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionShow,
	}
	cmd.AddCommand(showCmd)

	return cmd
}

func runSessionList(cmd *cobra.Command, args []string) error {
	cfg := getConfig(cmd)
	dataDir := cfg.Options.DataDirectory
	if dataDir == "" {
		dataDir = ".techne"
	}

	database, err := db.Open(dataDir + "/techne.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	store := db.NewSessionStore(database)

	sessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		cmd.Println("No sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tCREATED\tMESSAGES")

	for _, sess := range sessions {
		messages, err := store.GetMessages(sess.ID)
		if err != nil {
			messages = nil
		}
		created := sess.CreatedAt.Format("2006-01-02 15:04")
		title := sess.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", sess.ID[:8], title, created, len(messages))
	}

	return w.Flush()
}

func runSessionDelete(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	cfg := getConfig(cmd)
	dataDir := cfg.Options.DataDirectory
	if dataDir == "" {
		dataDir = ".techne"
	}

	database, err := db.Open(dataDir + "/techne.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	store := db.NewSessionStore(database)

	sess, err := store.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if sess == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	cmd.Printf("Session: %s\n", sess.Title)
	cmd.Printf("Created: %s\n", sess.CreatedAt.Format("2006-01-02 15:04:05"))

	messages, _ := store.GetMessages(sess.ID)
	cmd.Printf("Messages: %d\n", len(messages))

	cmd.Print("\nAre you sure you want to delete this session? [y/N]: ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		cmd.Println("Cancelled.")
		return nil
	}

	if err := store.DeleteSession(sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	cmd.Printf("Session %s deleted.\n", sessionID)
	return nil
}

func runSessionShow(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	cfg := getConfig(cmd)
	dataDir := cfg.Options.DataDirectory
	if dataDir == "" {
		dataDir = ".techne"
	}

	database, err := db.Open(dataDir + "/techne.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	store := db.NewSessionStore(database)

	sess, err := store.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if sess == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	messages, err := store.GetMessages(sess.ID)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	cmd.Printf("Session: %s\n", sess.ID)
	cmd.Printf("Title: %s\n", sess.Title)
	cmd.Printf("Model: %s\n", sess.Model)
	cmd.Printf("Provider: %s\n", sess.Provider)
	cmd.Printf("Created: %s\n", sess.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("Updated: %s\n", sess.UpdatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("\nMessages (%d):\n", len(messages))
	cmd.Println(strings.Repeat("-", 60))

	for i, msg := range messages {
		cmd.Printf("\n[%d] %s\n", i+1, strings.ToUpper(msg.Role))
		cmd.Printf("Time: %s\n", msg.CreatedAt.Format("15:04:05"))

		content := string(msg.Content)
		if len(content) > 500 {
			content = content[:497] + "..."
		}
		content = strings.TrimSpace(content)
		if content == "" {
			content = "(empty or structured content)"
		}

		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if len(line) > 100 {
				line = line[:97] + "..."
			}
			cmd.Printf("  %s\n", line)
		}
	}

	return nil
}

func init() {
	_ = strconv.Itoa(0)
	_ = os.Stdout
}
