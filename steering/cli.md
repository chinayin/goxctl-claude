---
inclusion: fileMatch
fileMatchPattern: "{cmd/**}"
---

# CLI Standards (cobra + gox/cli)

Use [spf13/cobra](https://github.com/spf13/cobra) for command-line parsing.
Use [gox/cli](https://github.com/chinayin/gox) for startup banner and parameter display.

## Framework

- CLI framework: `github.com/spf13/cobra`
- Startup output: `github.com/chinayin/gox/cli` + `gox/cli/cobra` adapter
- Never manually parse `os.Args` (use cobra for all subcommands and flags)

## Command Structure

```go
package main

import (
    "github.com/spf13/cobra"
    "github.com/chinayin/gox/cli"
    clicobra "github.com/chinayin/gox/cli/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
    Use:     "app-server",
    Version: version,
    RunE:    run,
}

func init() {
    rootCmd.Flags().StringVar(&configPath, "config", "config/server.yaml", "config file path")
}

func run(cmd *cobra.Command, args []string) error {
    adapter := clicobra.NewAdapter(cmd)
    cli.NewStartupWithAdapter(adapter).
        AutoAddFlags("help", "version").
        AddSection(
            cli.NewSection("Configuration").
                Add("Config", configPath),
        ).
        AddEndpoint("HTTP", cfg.Server.HTTP).
        AddEndpoint("gRPC", cfg.Server.GRPC).
        Print()

    // ... start server
    return nil
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Subcommand Pattern

Subcommands use separate files within the same `cmd/<app>/` directory:

```
cmd/server/
├── main.go          # rootCmd + main()
├── config.go        # config struct definition
├── run.go           # main service logic (rootCmd.RunE)
└── ca.go            # ca subcommand (ca init / ca issue)
```

```go
// ca.go
var caCmd = &cobra.Command{
    Use:   "ca",
    Short: "Certificate authority management",
}

var caInitCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize CA root certificate",
    RunE: func(cmd *cobra.Command, args []string) error {
        return pki.InitCA(caDir)
    },
}

func init() {
    caCmd.AddCommand(caInitCmd, caIssueCmd)
    rootCmd.AddCommand(caCmd)
}
```

## Flag Conventions

| Rule | Example |
|------|---------|
| Global flags use PersistentFlags | `rootCmd.PersistentFlags().StringVar(&configPath, "config", ...)` |
| Subcommand flags use Flags | `caIssueCmd.Flags().StringVar(&name, "name", ...)` |
| Flag names use kebab-case | `--ca-dir`, `--node-id` |
| Short flags only for high-frequency flags | `-c` for `--config` |
| Bool flags without value | `--server` (not `--server=true`) |

## Startup Banner

All microservices must print a banner on startup (using gox/cli):

```
MyApp v1.0.0
--------------------------------------------------------------------------------

Configuration
  Config:              config/server.yaml

Server Endpoints
  HTTP:                :8080
  gRPC:                :9090

--------------------------------------------------------------------------------
Server started successfully
  Press Ctrl+C to shutdown gracefully
```

- Use `cli.NewStartupWithAdapter(clicobra.NewAdapter(cmd))` to auto-extract app name and version
- `AutoAddFlags("help", "version")` excludes meaningless flags
- Only display flags with non-default values (changed flags)

## Version Info

Inject version information via ldflags:

```go
// version.go
package main

var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)
```

```makefile
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
```

## Error Handling

- Use `RunE` (not `Run`) — return error for cobra to handle uniformly
- Never call `os.Exit` inside RunE, let main handle exit codes
- cobra automatically prints Usage when user inputs invalid commands

## Local Config Repository (conditional, CLI projects managing local state)

When a CLI project needs to manage a local config repository, declare in core package `const.go`:

```go
const (
    AppName              = "<app-name>"
    DefaultConfigRepoURL = "ssh://git@git.example.com/op/config/<app-name>-config.git"
    ConfigDirName        = ".<app-name>"
)
```

- Do not hardcode project identifiers in business code
- Manage config repository addresses through centralized constants
