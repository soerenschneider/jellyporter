# 🌀 jellyporter

**jellyporter** is an application that syncs user playback data (UserData) — such as watched status, resume position, and playback timestamps — for Jellyfin items (episodes and movies) across multiple Jellyfin servers.

---

## ✨ Features

- ✅ Syncs UserData across multiple Jellyfin instances
- 📄 Configurable via a simple YAML file
- 🔀 Works with multiple users and servers


- 🔄 **Delta Syncing**  
  Efficiently syncs only changed items for minimal API usage and very fast updates and can therefore be run frequently.

- 🔔 **Event-Driven Sync**  
  Supports external event sources (e.g., webhooks) to trigger real-time synchronization.

- 🧠 **Smart Matching**  
  Identifies which Jellyfin instances require updates by comparing item `ProviderIDs` across servers.

- 📦 **Single Binary or Docker Image**  
  Easily deployable as a standalone binary or via Docker with no external dependencies.

- ⚙️ **Clean YAML Configuration**  
  Simple and readable YAML-based config with validation for common mistakes.

- 🕵️ **Audit Logging**  
  Maintains a detailed log of sync actions for traceability and debugging.

- 📊 **OpenTelemetry Metrics Support**  
  Exposes metrics for easy monitoring and alerting.

- 🚀 **Fast Startup**  
  Optimized for minimal startup time, even with large item sets or many clients.

- 🔒 **Read-Only Operation**  
  The tool never modifies your Jellyfin content—only updates user data (e.g., played/unplayed status).


---

## 📦 Configuration

Create a `config.yaml` file like this:

### General Config Structure

```yaml
database:
  path: /path/to/local.db

clients:
  my-jellyfin:
    url: http://localhost:8096
    user: myusername
    api_key: myapikey

events:
  webhook:
    addr: "0.0.0.0:9000"
    path: "/webhook"

sync_interval_mins: 5
full_sync_interval_mins: 360

metrics_addr: "127.0.0.1:8972"
metrics_path: "/metrics"
```

## Fields

### database.path
- Description: Path to the local database file.
- Type: string
- Validation: Optional. Must be a valid file path.

### clients
- Description: One or more Jellyfin server configurations.
- Type: map[string]JellyfinServerConfig
- Validation: Each entry must include a valid URL, username, and API key.

#### JellyfinServerConfig

| Field    | Description         | Validation            |
|----------|---------------------|------------------------|
| url      | Jellyfin server URL | Must be valid HTTP URL |
| user     | Jellyfin username   | Alphanumeric only      |
| api_key  | Jellyfin API key    | Alphanumeric only      |

### events.webhook
- Description: Optional webhook server to listen for events that trigger syncs.
- Type: struct
- Fields:
    - addr: Address to bind the webhook server (e.g., 0.0.0.0:9000)
    - path: Path to accept incoming webhooks (e.g., /webhook)

### sync_interval_mins
- Description: Interval (in minutes) for regular (incremental) synchronization.
- Default: 5
- Minimum: 5

### full_sync_interval_mins
- Description: Interval (in minutes) for full sync operations.
- Default: 360 (6 hours)
- Minimum: 30

### metrics_addr
- Description: Address to expose Prometheus metrics.
- Default: 127.0.0.1:8972
- Validation: Must be a valid host:port format.

### metrics_path
- Description: Optional custom path for exposing metrics.
- Default: /metrics (if omitted)
- Validation: Optional valid file path.

## Defaults

| Field                     | Default Value       |
|--------------------------|---------------------|
| sync_interval_mins       | 5                   |
| full_sync_interval_mins  | 360                 |
| metrics_addr             | 127.0.0.1:8972      |

## Validation Notes

- All fields are validated using go-playground/validator (https://github.com/go-playground/validator).
- Invalid configurations will result in startup errors with validation messages.
