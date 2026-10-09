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
    duration: <seconds, or auto>
    display: <led-catcher URL>           # optional, for duration: auto
    color: <local color/palette, or a device palette name>
    palette: <device palette name or ID>   # optional
    segments:
      - <segment ID>                      # optional
    speed: <0-255>                        # optional, default 128
    intensity: <0-255>                    # optional, default 255
    brightness: <0-255>                   # optional
    transition: <seconds>                 # optional
    restore: <true|false>                 # optional
    endpoint: <WLED HTTP endpoint>
```

## Fields

| Field | Description |
|-------|-------------|
| `systems` | List of source systems to match (e.g., `github`, `gitlab`), case-insensitively. Use `"*"` for all. |
| `severity` | List of severity levels to match (e.g., `ERROR`, `WARNING`, `INFO`, `SUCCESS`). |
| `tags` | *Optional.* Tags that must **all** be present in the message's comma-separated `tags` field, each matching one whole element (e.g., `[transition=point, side=a]`). See [Matching on tags](#matching-on-tags). |
| `fx` | WLED effect name (e.g., `Blurz`, `DJ Light`, `Aurora`, `Twinkle`), looked up case-insensitively in the device's `/json/eff` -- any effect the device has works. A number (e.g., `23`) is used as the effect ID directly. |
| `duration` | How long the effect runs in seconds before turning off -- or, with `restore`, before the previous scene comes back. `auto` keeps the light on exactly as long as the led-catcher's LED matrix shows something, see [Following the LED matrix](#following-the-led-matrix). |
| `display` | *Optional.* The led-catcher base URL `duration: auto` follows (e.g. `http://homerun2-led-catcher`). Without it, `LED_CATCHER_URL` is used; with neither, an `auto` effect is not sent. |
| `color` | A local color or palette (e.g., `red`, `sunset`, `ocean`, `forest`, `beach`), sent as the segment's colors. A name that is not local is looked up case-insensitively in the device's `/json/pal` and sent as a WLED palette (e.g., `Lava`, `Rainbow`) -- the device's colors stay. A local name wins over a device palette of the same name. |
| `palette` | *Optional.* A WLED palette by name (from the device's `/json/pal`) or by ID, sent **in addition** to `color`. With `palette` set, `color` must be a local name. Either `color` or `palette` is required. |
| `segments` | *Optional.* WLED segment IDs (0-31) the effect goes to. Without it the effect goes to the device's main segment. |
| `speed` | *Optional.* Effect speed (`sx`), 0-255. Default 128. |
| `intensity` | *Optional.* Effect intensity (`ix`), 0-255. Default 255. |
| `brightness` | *Optional.* Master brightness (`bri`), 0-255. Without it the device keeps its brightness. |
| `transition` | *Optional.* Crossfade into the effect in seconds (WLED `tt`, 0.1 s steps, max 6553.5). |
| `restore` | *Optional.* When `true`, the device's state is read before the effect and written back when `duration` ends, instead of switching the light off -- for devices that show an ambient scene. Effects that follow each other before the restore restore the scene from before the first one. If the state cannot be read, the light is switched off as without `restore`. |
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

## Quiet hours

`quietHours` keeps the strip calm outside office hours: from `from` to `to`
(it may cross midnight), and all of Saturday and Sunday with `weekends: true`,
only the `allow` severities match an effect (default `error`, `critical`).
Everything else matches nothing and the strip stays as it is. Times are
`HH:MM` in `timezone` (an IANA name; empty is the host's local time). An
invalid time or zone is rejected when the profile is loaded. The profile is
re-read for every message, so a change applies without a restart.

```yaml
quietHours:
  from: "19:00"
  to: "07:00"
  weekends: true
  timezone: Europe/Berlin
  allow: [error, critical]
effects:
  ...
```

The schema is the same as led-catcher's `quietHours`, so one profile reads the
same on both catchers.

## Following the LED matrix

The led-catcher scrolls a text for (64 + text width) x 30 ms -- a width only it knows, from its own profile and font -- and drops messages that arrive while the panel is busy. A fixed `duration` therefore drifts apart from the panel in every burst of messages.

With `duration: auto` the light-catcher asks the panel instead: it polls the led-catcher's `GET /display` (no token needed) every 250 ms and ends the effect -- off, or back to the previous scene with `restore` -- when the panel stops showing:

```yaml
effects:
  alert:
    systems: ["*"]
    severity: [error]
    fx: Fireworks
    color: red
    duration: auto
    display: http://homerun2-led-catcher
    endpoint: http://wled
```

- If the panel is still busy with an earlier message when this one arrives, the led-catcher drops this one and the light stays with what is shown: light and panel end together.
- If the panel does not start showing within 3 s (the led-catcher matched nothing), or cannot be reached for 5 s, the effect ends.
- A held display, or a panel that never finishes, is capped at 2 minutes.

## WLED Mock

For development and testing, use the embedded WLED mock server:

```bash
MOCK_WLED=true MOCK_WLED_PORT=9090 go run .
```

Like a real device, the mock serves `/json/eff` and `/json/pal` and merges each `POST /json/state` into its state (a palette-only update keeps the colors, `{"on":false}` keeps the segments), so `restore` and segments behave as on hardware. `WLED_PALETTES_FILE` points it at a palette list captured from a device (`curl http://<wled>/json/pal > palettes.json`).

The mock provides an HTML dashboard at `http://localhost:9090` showing received effects in real time.

## Dashboard

The dashboard (`:8080/`) lists the effects the catcher played, newest first.
Each row shows the message that triggered it: severity, system and title.
A click expands the message text, author, all message tags, the link (PR,
report, alert) and the WLED endpoint, so you can trace what made the strip
light up. Every field is escaped, and only `http(s)` URLs become links.

The ▶ button plays that effect again: the event's severity, system and
message tags are matched against the profile as it is now, **without quiet
hours**, because a click is a deliberate request. The replay goes into the
timeline as a new row, marked ↻. `POST /api/events/{id}/replay` is the
endpoint behind it: 202, 404 for an unknown id or a "light turned off" row,
409 when the profile matches no effect any more, 429 above 30 per minute.
