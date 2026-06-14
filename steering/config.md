---
inclusion: fileMatch
fileMatchPattern: "{config/**,cmd/**/main.go,cmd/**/config.go,**/bootstrap/**}"
---

# Configuration Management Standards (gox/config)

Use [github.com/chinayin/gox/config](https://github.com/chinayin/gox) exclusively for configuration loading.
Never use raw `os.Getenv` for structured config, never use viper directly in business code.

## Loading Mechanism

gox/config is built on Viper. Loading priority from low to high:

1. struct tag `default:"value"` — in-code defaults
2. Main config file `config/<app>.yaml` — project convention
3. Local override `config/<app>.local.yaml` — gitignored, for dev environment
4. Environment variables — auto-mapped, used for Docker/K8s overrides

```go
loader := config.NewLoader()
var cfg AppConfig
if err := loader.Load("config/server.yaml", &cfg); err != nil {
    slog.Error("load config failed", "error", err)
    os.Exit(1)
}
```

## Struct Definition Rules (MUST)

### Tag Requirements

viper Unmarshal uses mapstructure (not yaml tag) for field mapping. Rules:

- If Go field name matches YAML key (case-insensitive): **no tags needed**
- If Go field name differs from YAML key: **`mapstructure` tag is required**
- Always add `yaml` tag when `mapstructure` is present (for non-viper YAML parsers)

```go
// Good — field name matches key, no tags needed
type Config struct {
    Server struct {
        HTTP string `default:":8080"`
        GRPC string `default:":9090"`
    }
    Database struct {
        Path string `default:"data/app.db"`
    }
}

// Good — field name differs from key, mapstructure required
type AgentConfig struct {
    Agent struct {
        NodeID string `yaml:"id" mapstructure:"id"`  // NodeID != "id"
        Server string                                 // Server == "server", no tag needed
        Token  string                                 // Token == "token", no tag needed
    }
}

// Bad — field name differs but mapstructure missing, viper will lose values
type AgentConfig struct {
    Agent struct {
        NodeID string `yaml:"id"`  // viper cannot map "id" to NodeID without mapstructure
    }
}
```

### Field Naming

YAML keys must not use snake_case (viper's `SetEnvKeyReplacer(".", "_")` creates ambiguity):

```yaml
# Good — single words or abbreviations
agent:
  id: "node-1"
  server: "localhost:9090"
  token: ""

# Bad — underscores conflict with hierarchy separator
agent:
  node_id: "node-1"       # env AGENT_NODE_ID resolves to agent.node.id
  server_addr: "localhost" # env AGENT_SERVER_ADDR → agent.server.addr
```

### Default Values

- Fields with sensible defaults use `default` tag
- Sensitive fields (token/password/key path) must not have defaults, forcing explicit configuration
- Optional feature toggles default to false/empty (not enabled unless configured)

## Environment Variable Mapping

Viper auto-mapping rule: `yaml.path.key` → `YAML_PATH_KEY`

```
server.http  → SERVER_HTTP
agent.id     → AGENT_ID
agent.token  → AGENT_TOKEN
database.path → DATABASE_PATH
```

### When to Use Environment Variables

| Scenario | Use env var | Use config file |
|----------|-------------|-----------------|
| Passwords/Tokens/Secrets | Yes | No |
| Certificate file paths (K8s Secret mount) | Yes | Yes |
| Ports/Addresses (differ per environment) | Yes (override) | Yes (defaults) |
| Engine type/strategy parameters | No | Yes |
| Complex structures (labels/backends) | No | Yes |

### Special Environment Variables Outside Config Struct

Some environment variables are read directly by specific modules (e.g., security validation), not through the config struct:

```go
// These envs are read via os.Getenv inside the module, not placed in config struct
const envRegisterToken = "REGISTER_TOKEN"  // used by agent token validator
```

Such variables must be clearly documented in the corresponding package comments.

## File Structure (MUST)

```
config/
├── server.yaml          # server main config
├── server.local.yaml    # server local override (gitignored)
├── agent.yaml           # agent main config
└── agent.local.yaml     # agent local override (gitignored)
```

- `.local.yaml` must be in `.gitignore`
- Main config files are committed to the repository with all fields and comments

## CLI Parameter (MUST)

All cmd entry points must support `--config=<path>` parameter to specify config file path.

Default path conventions:
- Single-app project: `config/config.yaml`
- Multi-app project: `config/<app>.yaml` (e.g., `config/server.yaml`, `config/agent.yaml`)

For implementation details using cobra flags, see the `cli` steering file.

## Docker / K8s Deployment

```yaml
# K8s: config files via ConfigMap, secrets via Secret
spec:
  containers:
    - name: app-server
      volumeMounts:
        - name: config
          mountPath: /app/config/server.yaml
          subPath: server.yaml
        - name: tls
          mountPath: /app/certs
      env:
        - name: REGISTER_TOKEN
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: register-token
```

## Common Pitfalls

### viper environment variable override not working

Cause: when struct field name differs from yaml key, `mapstructure` tag is required. viper Unmarshal looks up mapstructure tag, not yaml tag.

### Environment variable set but value is empty

Cause: YAML key contains underscores, `AGENT_NODE_ID` is mapped to `agent.node.id` instead of `agent.node_id`. Solution: do not use underscores in YAML keys.

### .local.yaml override not working

Cause: the field is set to empty string `""` in main yaml, viper considers it already has a value and skips override. Solution: do not write the field in main yaml, or comment it out.
