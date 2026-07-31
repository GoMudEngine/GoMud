# Discord Integration Context

## Purpose

`internal/integrations/discord` mirrors selected in-game events to a Discord
webhook: logins, deaths, broadcasts, auction updates, and server logs. It is
outbound-only — nothing comes back from Discord into the game.

## Files

- **client.go** — `Init`, the send path, and request backoff.
- **listeners.go** — one handler per mirrored event.
- **types.go** — `Color` and the webhook payload/embed shapes.

## API

```go
func Init(webhookUrl string)
func SendMessage(message string)
func SendRichMessage(message string, color Color)
func SendPayload(payload webHookPayload)
```

`Init` registers the listeners. If the webhook URL is empty the integration
stays dormant.

## Mirrored events

`HandlePlayerSpawn`, `HandlePlayerDespawn`, `HandleDeath`, `HandleBroadcast`,
`HandleAuctionUpdate`, `HandleLogs` — each an `events.Event` listener returning
`events.ListenerReturn`.

## Gotchas

- **This sends game data to a third party.** Anything routed here leaves the
  server. Do not add a listener that mirrors player-private content — tells,
  petitions, moderation notes — without an explicit decision to publish it.
- **`HandleLogs` can be noisy and can leak internals.** Server log lines were
  never written for a public audience; keep the level filter tight.
- **Sends are rate-limited by `isRequestBackoff` / `doRequestBackoff`.** A burst
  of deaths will be throttled, so Discord is not a reliable audit trail —
  messages can be dropped.
- **Failures are logged, not retried.** A webhook outage loses those events.

## Dependencies

`events`, `configs`, `mudlog`, plus `net/http` and JSON.

## Consumers

Registered from the engine start-up path when a webhook URL is configured.
