package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/romerox3/volra/internal/templates"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart [template] [directory]",
	Short: "Create a new agent project from a template",
	Long:  "Scaffold a new agent project from a built-in template. Run without arguments for interactive mode.",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runQuickstart,
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}

// dnsNameRe validates project names as DNS-safe identifiers.
var dnsNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func runQuickstart(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runQuickstartInteractive()
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: volra quickstart <template> <directory>")
	}

	return scaffold(args[0], args[1])
}

func runQuickstartInteractive() error {
	// Check if stdin is a terminal for interactive mode.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive: just list templates.
		return listTemplates()
	}

	available := templates.Available()
	scanner := bufio.NewScanner(os.Stdin)

	// Step 1: Choose template
	fmt.Println("Choose a template:")
	currentCategory := ""
	for i, t := range available {
		if t.Category != currentCategory {
			currentCategory = t.Category
			fmt.Printf("\n  %s:\n", currentCategory)
		}
		fmt.Printf("  %2d) %-20s %s\n", i+1, t.Name, t.Description)
	}
	fmt.Println()
	fmt.Printf("Enter number (1-%d): ", len(available))

	if !scanner.Scan() {
		return fmt.Errorf("cancelled")
	}
	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(available) {
		return fmt.Errorf("invalid choice: %s", scanner.Text())
	}
	templateName := available[choice-1].Name

	// Step 2: Project name
	fmt.Print("Project name (DNS-safe, e.g. my-agent): ")
	if !scanner.Scan() {
		return fmt.Errorf("cancelled")
	}
	projectName := strings.TrimSpace(scanner.Text())
	if !dnsNameRe.MatchString(projectName) {
		return fmt.Errorf("invalid project name %q — must be lowercase, start with a letter, and contain only a-z, 0-9, hyphens", projectName)
	}

	fmt.Println()
	return scaffold(templateName, projectName)
}

func listTemplates() error {
	fmt.Println("Available templates:")
	currentCategory := ""
	for _, t := range templates.Available() {
		if t.Category != currentCategory {
			currentCategory = t.Category
			fmt.Printf("\n  %s:\n", currentCategory)
		}
		fmt.Printf("    %-20s %s\n", t.Name, t.Description)
	}
	fmt.Println()
	fmt.Println("Usage: volra quickstart <template> <directory>")
	return nil
}

func scaffold(templateName, targetDir string) error {
	// Check if target directory already exists with content.
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(targetDir)
		if len(entries) > 0 {
			return fmt.Errorf("directory %q already exists and is not empty", targetDir)
		}
	}

	projectName := filepath.Base(targetDir)

	if err := templates.Scaffold(templateName, targetDir, projectName); err != nil {
		return err
	}

	fmt.Printf("Created %s project in %s/\n", templateName, targetDir)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", targetDir)
	fmt.Println("  volra deploy")

	return nil
}
