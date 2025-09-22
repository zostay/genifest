package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zostay/genifest/internal/config"
)

// ValidationSummaryError is a special error type that indicates validation failed
// but the summary and error display have already been handled.
type ValidationSummaryError struct {
	ErrorCount int
}

func (e *ValidationSummaryError) Error() string {
	return fmt.Sprintf("validation failed with %d error(s)", e.ErrorCount)
}

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

Schema validation modes:
- Default: Permissive mode (ignore unknown fields)
- --warn: Show warnings for unknown fields but continue
- --strict: Fail validation on unknown fields

If a directory is specified, the command will operate from that directory instead 
of the current working directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var projectDir string
		if len(args) > 0 {
			projectDir = args[0]
		}

		// Determine validation mode from flags
		strict, _ := cmd.Flags().GetBool("strict")
		warn, _ := cmd.Flags().GetBool("warn")

		var mode config.ValidationMode
		if strict && warn {
			fmt.Fprintf(os.Stderr, "❌ Cannot use both --strict and --warn flags together\n")
			os.Exit(1)
		} else if strict {
			mode = config.ValidationModeStrict
		} else if warn {
			mode = config.ValidationModeWarn
		} else {
			mode = config.ValidationModePermissive
		}

		err := validateConfigurationWithMode(projectDir, mode)
		if err != nil {
			// Check if it's our special validation summary error (handled by validateConfiguration)
			var summaryErr *ValidationSummaryError
			if errors.As(err, &summaryErr) {
				// Error has been handled, just exit with error code
				os.Exit(1)
			} else if isValidationError(err) {
				// ValidationError from cfg.Validate() - these should be handled by validateConfiguration
				// but if they reach here, we need to show them properly
				fmt.Fprintf(os.Stderr, "❌ Configuration validation failed\n")
				os.Exit(1)
			} else {
				// Other errors need normal handling
				printError(err)
			}
		}
	},
}

func init() {
	validateCmd.Flags().Bool("strict", false, "Enable strict schema validation (fail on unknown fields)")
	validateCmd.Flags().Bool("warn", false, "Enable schema validation warnings (warn on unknown fields)")
	rootCmd.AddCommand(validateCmd)
}

func validateConfiguration(projectDir string) error {
	return validateConfigurationWithMode(projectDir, config.ValidationModePermissive)
}

func validateConfigurationWithMode(projectDir string, mode config.ValidationMode) error {
	// Load project configuration - if this fails with ValidationError, we need special handling
	projectInfo, err := loadProjectConfigurationWithMode(projectDir, mode)
	if err != nil {
		// Check if it's a ValidationError from config loading
		var ve *config.ValidationError
		if errors.As(err, &ve) {
			// Show at least the basic info before the error
			workDir, _ := resolveProjectDirectory(projectDir)
			fmt.Printf("🔍 \033[1;34mValidating configuration in %s...\033[0m\n\n", workDir)

			// Extract the clean message without the emoji prefix
			msg := strings.TrimPrefix(ve.Error(), "❌ ")

			fmt.Printf("❌ \033[1;31mConfiguration validation failed with 1 error:\033[0m\n\n")
			fmt.Printf("  \033[31m•\033[0m %s\n", msg)
			return &ValidationSummaryError{ErrorCount: 1}
		}
		return err
	}

	workDir := projectInfo.WorkDir
	cfg := projectInfo.Config

	fmt.Printf("🔍 \033[1;34mValidating configuration in %s...\033[0m\n\n", workDir)

	// Collect all validation errors instead of short-circuiting
	validationErrors := []string{}

	// Run comprehensive validation from config package and collect errors
	if err := cfg.Validate(); err != nil {
		// Handle ValidationError specially to extract just the message
		var ve *config.ValidationError
		if errors.As(err, &ve) {
			// Extract the clean message without the emoji prefix
			msg := strings.TrimPrefix(ve.Error(), "❌ ")
			validationErrors = append(validationErrors, msg)
		} else {
			validationErrors = append(validationErrors, err.Error())
		}
	}

	// Resolve files and check if they exist
	resolvedFiles, err := cfg.Files.ResolveFiles(workDir)
	if err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("failed to resolve file patterns: %s", err.Error()))
		resolvedFiles = cfg.Files.Include // Fallback for further validation
	}

	for _, filePath := range resolvedFiles {
		fullPath := filepath.Join(workDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			validationErrors = append(validationErrors, fmt.Sprintf("referenced file does not exist: %s", filePath))
		}
	}

	// Additional validation checks beyond what Config.Validate() already does
	// Check for duplicate function names (Config.Validate() doesn't check for this)
	functionNames := make(map[string]bool)
	for i, fn := range cfg.Functions {
		if fn.Name != "" && functionNames[fn.Name] {
			validationErrors = append(validationErrors, fmt.Sprintf("function %d: duplicate function name '%s'", i, fn.Name))
		}
		if fn.Name != "" {
			functionNames[fn.Name] = true
		}
	}

	// Always show summary information first
	fmt.Printf("\033[1mSummary:\033[0m\n")
	// Get resolved files count for display (reuse resolvedFiles from above)
	if len(resolvedFiles) == 0 && len(cfg.Files.Include) > 0 {
		resolvedFiles = cfg.Files.Include // Fallback
	}
	fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m file(s) managed\n", len(resolvedFiles))
	fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m change(s) defined\n", len(cfg.Changes))
	fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m function(s) defined\n", len(cfg.Functions))

	// Show tags if any
	tagSet := make(map[string]bool)
	for _, change := range cfg.Changes {
		if change.Tag != "" {
			tagSet[change.Tag] = true
		}
	}
	if len(tagSet) > 0 {
		fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m unique tag(s) used\n", len(tagSet))
	}
	fmt.Println()

	// Report results
	if len(validationErrors) > 0 {
		fmt.Printf("❌ \033[1;31mConfiguration validation failed with %d error(s):\033[0m\n\n", len(validationErrors))
		for _, err := range validationErrors {
			fmt.Printf("  \033[31m•\033[0m %s\n", err)
		}
		return &ValidationSummaryError{ErrorCount: len(validationErrors)}
	}

	// Success message
	fmt.Printf("✅ \033[1;32mConfiguration validation successful!\033[0m\n")

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
