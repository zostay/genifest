# Troubleshooting

Common issues and solutions when using Genifest.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Installation Issues

### Command Not Found

**Problem**: `genifest: command not found`

**Solutions**:
```bash
# Check if genifest is in PATH
which genifest

# Add to PATH if needed
export PATH="/usr/local/bin:$PATH"

# Or reinstall
curl -L https://raw.githubusercontent.com/zostay/genifest/master/tools/install.sh | sh
```

### Permission Denied (macOS)

**Problem**: `"genifest" cannot be opened because it is from an unidentified developer`

**Solution**:
```bash
# Remove quarantine attribute
sudo xattr -d com.apple.quarantine /usr/local/bin/genifest
```

## Configuration Issues

### Configuration File Not Found

**Problem**: `Configuration file not found`

**Solution**: Ensure you're in a directory with `genifest.yaml`:
```bash
ls genifest.yaml
# Or specify directory
genifest run /path/to/project
```

### No Changes Applied

**Problem**: `0 change(s) applied` despite having changes defined

**Possible causes**:
1. **File selectors don't match files**:
   ```bash
   genifest config  # Check merged configuration
   ```

2. **Tag filtering excludes changes**:
   ```bash
   genifest tags    # See available tags
   genifest run     # Run without tag filters
   ```

3. **Key selectors don't match YAML structure**:
   ```bash
   # Check YAML structure matches keySelector
   ```

### Function Not Found

**Problem**: `Function 'function-name' not found`

**Solutions**:
1. **Check function definition**:
   ```bash
   genifest config | grep -A5 functions
   ```

2. **Verify function scope**: Functions are only available in their definition directory and children

3. **Check spelling**: Function names are case-sensitive

## Runtime Issues

### Script Execution Fails

**Problem**: Scripts fail to execute

**Solutions**:
1. **Check script permissions**:
   ```bash
   chmod +x scripts/script-name.sh
   ```

2. **Verify script path**: Scripts must be in configured script directories

3. **Check script errors**: Scripts should exit with status 0

### File Not Found

**Problem**: `File not found` when using file inclusion

**Solutions**:

1. **Check file path**: Files must be in configured file directories
2. **Verify file exists**: Use relative paths from the files directory
3. **Check permissions**: Ensure files are readable

## Validation Errors

### Invalid YAML

**Problem**: YAML parsing errors

**Solutions**:

1. **Check YAML syntax**:
   ```bash
   # Use a YAML validator
   yamllint genifest.yaml
   ```

2. **Check indentation**: YAML is sensitive to indentation
3. **Escape special characters**: Quote strings with special characters

### Path Security Violations

**Problem**: `Path outside cloudHome boundary`

**Solution**: All paths must be within the configured `cloudHome` directory

## Performance Issues

### Slow Processing

**Problem**: Genifest runs slowly

**Solutions**:

1. **Reduce file count**: Use more specific file selectors
2. **Optimize functions**: Avoid complex template operations
3. **Check script performance**: Scripts should execute quickly

## Getting Help

If you can't resolve an issue:

1. **Check logs**: Run with verbose output if available
2. **Validate configuration**: Use `genifest validate`
3. **Report bugs**: [GitHub Issues](https://github.com/zostay/genifest/issues)
4. **Ask questions**: [GitHub Discussions](https://github.com/zostay/genifest/discussions)

## See Also

- [Installation Guide](../getting-started/installation.md) - Installation instructions
- [CLI Reference](../user-guide/cli-reference.md) - Command documentation
- [Configuration Guide](../user-guide/configuration.md) - Configuration reference