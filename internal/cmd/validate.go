package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zostay/genifest/internal/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate [directory]",
	Short: "Validate the genifest configuration",
	Long: `Validate the genifest configuration files for syntax errors, 
missing dependencies, and other configuration issues.

This command will:
- Load and parse all configuration files
- Validate function references and dependencies
- Check path security and cloudHome boundaries
- Verify file selectors and key selectors
- Report any configuration errors found

If a directory is specified, the command will operate from that directory instead 
of the current working directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var projectDir string
		if len(args) > 0 {
			projectDir = args[0]
		}
		err := validateConfiguration(projectDir)
		if err != nil {
			printError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func validateConfiguration(projectDir string) error {
	// Load project configuration
	projectInfo, err := loadProjectConfiguration(projectDir)
	if err != nil {
		return err
	}

	workDir := projectInfo.WorkDir
	cfg := projectInfo.Config

	fmt.Printf("Validating configuration in %s...\n\n", workDir)

	// Run comprehensive validation from config package
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Additional validation checks
	validationErrors := []string{}

	// Check if all referenced files exist
	for _, filePath := range cfg.Files {
		fullPath := filepath.Join(workDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			validationErrors = append(validationErrors, fmt.Sprintf("referenced file does not exist: %s", filePath))
		}
	}

	// Validate changes have proper selectors
	for i, change := range cfg.Changes {
		if change.FileSelector == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("change %d: fileSelector is required", i))
		}
		if change.KeySelector == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("change %d: keySelector is required", i))
		}
		if isValueFromEmpty(change.ValueFrom) {
			validationErrors = append(validationErrors, fmt.Sprintf("change %d: valueFrom is required", i))
		}
	}

	// Validate function definitions
	functionNames := make(map[string]bool)
	for i, fn := range cfg.Functions {
		if fn.Name == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("function %d: name is required", i))
		} else {
			if functionNames[fn.Name] {
				validationErrors = append(validationErrors, fmt.Sprintf("function %d: duplicate function name '%s'", i, fn.Name))
			}
			functionNames[fn.Name] = true
		}
		if isValueFromEmpty(fn.ValueFrom) {
			validationErrors = append(validationErrors, fmt.Sprintf("function %d (%s): valueFrom is required", i, fn.Name))
		}
	}

	// Report results
	if len(validationErrors) > 0 {
		fmt.Printf("❌ Configuration validation failed with %d error(s):\n\n", len(validationErrors))
		for _, err := range validationErrors {
			fmt.Printf("  • %s\n", err)
		}
		fmt.Printf("\n💡 Tip: Fix these issues and run 'genifest validate' again\n")
		return fmt.Errorf("configuration validation failed")
	}

	// Success summary
	fmt.Printf("✅ Configuration validation successful!\n\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  • %d file(s) managed\n", len(cfg.Files))
	fmt.Printf("  • %d change(s) defined\n", len(cfg.Changes))
	fmt.Printf("  • %d function(s) defined\n", len(cfg.Functions))

	// Show tags if any
	tagSet := make(map[string]bool)
	for _, change := range cfg.Changes {
		if change.Tag != "" {
			tagSet[change.Tag] = true
		}
	}
	if len(tagSet) > 0 {
		fmt.Printf("  • %d unique tag(s) used\n", len(tagSet))
	}

	return nil
}

// isValueFromEmpty checks if a ValueFrom struct has no fields set.
func isValueFromEmpty(vf config.ValueFrom) bool {
	return vf.FunctionCall == nil &&
		vf.CallPipeline == nil &&
		vf.FileInclusion == nil &&
		vf.BasicTemplate == nil &&
		vf.ScriptExec == nil &&
		vf.ArgumentRef == nil &&
		vf.DefaultValue == nil &&
		vf.DocumentRef == nil
}
