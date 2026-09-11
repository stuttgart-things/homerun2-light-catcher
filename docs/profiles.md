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
| `tags` | *Optional.* Tags that must **all** be present in the message's comma-separated `tags` field, each matching one whole element (e.g., `[transition=point, side=a]`). See [Matching on tags](#matching-on-tags). |
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

1. Extracts `system`, `severity` and `tags` from the message payload
2. Iterates through the profile effects **in the order they appear in the file**
3. Selects the first effect where the system matches (or `"*"`), severity matches (case-insensitive) and — if the effect lists `tags` — every listed tag is present
4. Sends the corresponding WLED effect via HTTP API
5. Waits for the configured duration, then turns the effect off — unless a newer effect has been sent to the same endpoint since

### First match wins

Because the first matching rule wins, declare specific rules above wildcard rules they overlap with:

```yaml
effects:
  tabletennis-win:        # checked first
    systems: [tabletennis]
    severity: [success]
    fx: Fireworks
  success:                # only reached for systems other than tabletennis
    systems: ["*"]
    severity: [success]
    fx: Aurora
```

Swapping the two entries makes `success` match every system, including `tabletennis`, so `tabletennis-win` would never be selected.

### Overlapping effects on one endpoint

Each effect with a `duration` schedules a turn-off. A turn-off is skipped if a newer effect was sent to the same endpoint after the one that scheduled it, so a short effect followed by a longer one does not cut the longer one short:

| t | event | light |
|---|-------|-------|
| 0s | point, `duration: 3` | flash starts |
| 2s | match won, `duration: 10` | celebration starts |
| 3s | the point's timer fires | skipped — celebration keeps running |
| 12s | the celebration's timer fires | off |

### Matching on tags

A rule can also require message tags. `tags` is optional; a rule without it matches on `systems` and `severity` alone, exactly as before.

```yaml
effects:
  tabletennis-match:
    systems: [tabletennis]
    severity: [success]
    tags: [transition=match_won]
    fx: Fireworks
    duration: 10
    color: forest
  tabletennis-point-a:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=a]
    fx: Solid
    duration: 1
    color: blue
  info:                   # declared after the tag rules, catches the rest
    systems: ["*"]
    severity: [info]
    fx: DJ Light
    color: ocean
```

- **All listed tags must be present (AND).** `[transition=point, side=a]` matches only messages carrying both.
- **Each entry matches one whole element** of the message's comma-separated `tags` field — `side=a` matches `match=36c17b30,set=2,transition=point,side=a` but not `side=ab`. Whitespace around elements is ignored; the comparison is case-sensitive.
- Declare tag rules **above** the wildcard rules they overlap with (first match wins).

> **Not the same as the notification-catcher.** homerun2-notification-catcher's `tags_contain` is a *substring* match that *ORs* the entries in its list. Here, entries are *whole elements* and *all* of them must match. Don't copy the rule shape between the two catchers unchanged.

Both dashboards show the tags a triggered rule matched on in their event timeline.

## WLED Mock

For development and testing, use the embedded WLED mock server:

```bash
MOCK_WLED=true MOCK_WLED_PORT=9090 go run .
```

The mock provides an HTML dashboard at `http://localhost:9090` showing received effects in real time.
