# Effect Profiles

Light Catcher uses YAML profiles to map incoming messages to WLED light effects. Each effect entry matches on system and severity to determine which WLED effect, color palette, and duration to apply.

## Profile Structure

```yaml
effects:
  <effect-name>:
    systems:
      - <system-name or "*" for all>
    severity:
      - <severity-level>
    fx: <WLED effect name>
    duration: <seconds>
    color: <color palette name>
    segments:
      - <segment index>
    endpoint: <WLED HTTP endpoint>
```

## Fields

| Field | Description |
|-------|-------------|
| `systems` | List of source systems to match (e.g., `github`, `gitlab`). Use `"*"` for all. |
| `severity` | List of severity levels to match (e.g., `ERROR`, `WARNING`, `INFO`, `SUCCESS`). |
| `fx` | WLED effect name (e.g., `Blurz`, `DJ Light`, `Aurora`, `Twinkle`). |
| `duration` | How long the effect runs in seconds before turning off. |
| `color` | Color palette name (e.g., `sunset`, `ocean`, `forest`, `beach`). |
| `segments` | WLED segment indices to apply the effect to. |
| `endpoint` | WLED device HTTP endpoint (e.g., `http://192.168.1.100:80`). |

## Example Profile

```yaml
effects:
  error-git:
    systems:
      - gitlab
      - github
    severity:
      - ERROR
    fx: Blurz
    duration: 3
    color: sunset
    segments:
      - 0
    endpoint: http://wled-device:80

  info:
    systems:
      - "*"
    severity:
      - INFO
    fx: DJ Light
    duration: 3
    color: ocean
    segments:
      - 0
    endpoint: http://wled-device:80

  success:
    systems:
      - "*"
    severity:
      - SUCCESS
    fx: Aurora
    duration: 3
    color: forest
    segments:
      - 0
    endpoint: http://wled-device:80

  warning:
    systems:
      - "*"
    severity:
      - WARNING
    fx: Twinkle
    duration: 3
    color: beach
    segments:
      - 0
    endpoint: http://wled-device:80
```

## Matching Logic

When a message arrives from a Redis Stream, the LightHandler:

1. Extracts `system` and `severity` from the message payload
2. Iterates through all profile effects
3. Selects the first effect where the system matches (or `"*"`) and severity matches
4. Sends the corresponding WLED effect via HTTP API
5. Waits for the configured duration, then turns the effect off

## WLED Mock

For development and testing, use the embedded WLED mock server:

```bash
MOCK_WLED=true MOCK_WLED_PORT=9090 go run .
```

The mock provides an HTML dashboard at `http://localhost:9090` showing received effects in real time.
