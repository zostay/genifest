## WIP  TBD

 * Adding Arm64 builds for Linux.

## v0.1.2  2024-08-09

 * Upgraded to Ghost v0.6.2

## v0.1.1  2024-08-09

 * Fix a bug in `file` that caused it to files when `files_dir` was set.

## v0.1.0  2024-08-09

 * Added the `files_dir` setting to cluster configuration to set the location of the directory used to load files when calling the `files` function when templating.
 * Fixed a minor problem with secrets skipping. It didn't always work correctly when the `applyTemplate` function to template a second file.
 * Added the `ghost` section to cluster configuration for configuring `ghost` for secrets management.
 * Added `ghost.config` to cluster configuration to select the configuration file to use for working with secrets.
 * Added `ghost.keeper` to cluster configuration to select the secrets keeper to use for working with secrets.

## v0.0.2  2024-06-27

 * Fixing the install script.

## v0.0.1  2024-06-27

 * Adding binaries to the release.

## v0.0.0  2024-06-25

 * Initial release.
