# Dashy Reborn

Dashy Reborn is a small Go adaptation of [Dashy](https://github.com/Lissy93/dashy).
It reads a Dashy-compatible YAML config and renders a server-side dashboard without a frontend build step.

## Scope

- Go-based, lightweight, server-rendered alternative for simple Dashy dashboards
- Supports the most common `pageInfo`, `appConfig`, `sections`, `items`, `subItems`, and `pages` fields
- Local YAML hot reload
- Local asset serving and on-disk favicon caching

## Known Limitations

- Status checks are not implemented
- Theme support is limited compared to the original Dashy project
- Some Dashy fields are ignored or degraded
- Widgets are rendered as simple placeholders
- Remote config sources can be loaded at startup, but are not watched for changes

## Quick Start

```sh
go run . -config .\conf.sample.yml
```

Then open `http://127.0.0.1:8080`.

If you want remote icons and other external assets enabled:

```powershell
go run . -config .\conf.sample.yml -assets-mode auto
```

## Main Flags

- `-config`: path or URL to the main YAML config
- `-public`: Dashy-style `public` directory for local assets
- `-addr`: HTTP listen address
- `-watch`: local config polling interval, `0` disables watching
- `-strict`: fail on unknown YAML fields
- `-assets-mode`: `auto`, `internal-only`, `offline`
- `-favicon-cache-dir`: on-disk cache directory for remote favicons

## Notes

- Default asset mode is `internal-only`
- `icon: favicon` is cached locally on disk and then served by the app
- `/healthz` exposes basic runtime and reload information
