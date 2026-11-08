# Notification-service rewrite — design

Date: 2026-11-08
Status: approved

## Intent

Notification-service was written before user-service was redesigned and no longer
fits the platform:

- its code style (package layout, config, lifecycle, Kafka consumption, error
  handling, tests) differs from user-service, which is now the reference service;
- it does not understand the events user-service publishes today. User-service
  emits camelCase envelopes on topic `users` with an `event-type` Kafka header
  (`user.registration_requested`, `user.email_change_requested`, …), some carrying
  a confirmation token sealed with notification-service's public key. The current
  consumer expects the old snake_case `user.registered` with `login_url`, so
  registration and email-change confirmation emails are never sent.

The goal is a notification-service that looks and behaves like a sibling of
user-service and sends every email the user flows need.

## Goals

1. Restructure notification-service in the style of `../user-service`
   (see "Style to follow").
2. Consume the user-service events from topic `users` and send the emails in the
   table below, decrypting confirmation tokens where needed.
3. Keep sending the order confirmation email for `orders.confirmed`
   (order-service), in the new code style.
4. Small user-service change: add the user's first/last name to three events so
   their emails can greet the user by name.

## Non-goals

- No database in notification-service. Every event already carries the recipient
  address; no local user projection is needed.
- No change to order-service or to the `orders.confirmed` payload format.
- No frontend confirmation pages (links point at a configurable frontend URL that
  may not exist yet).
- No HTTP API, no outbox, no retries beyond what the Kafka consumer does.
- No change of email provider (stays SendGrid).

## Anti-goals

- A plaintext confirmation token must never be logged or appear in error messages.
- An email must never reveal to the owner of an address who else tried to use it
  (see `email_change_to_taken_address_attempted` below).
- One bad message must not block a partition forever: undecodable or permanently
  failing messages are skipped with a log line, like user-service's `ErrPermanent`.

## Emails

Topic `users` (producer: user-service, envelope `{eventId, eventType, timestamp,
version, data}`, `event-type` header set):

| Event type | Recipient | Email | Name in greeting |
|---|---|---|---|
| `user.registration_requested` | `data.email` | Confirm registration — link with token | `firstName` (present) |
| `user.registered` | `data.email` | Welcome, link to shop/login | `firstName` (present) |
| `user.duplicate_registration_attempted` | `data.email` | "Someone tried to register with your address; if it was you, log in or reset password" | `firstName` (**added**, owner's name) |
| `user.registration_rejected` | — | no email | — |
| `user.email_change_requested` | `data.newEmail` | Confirm new address — link with token | `firstName` (**added**) |
| `user.email_change_to_taken_address_attempted` | `data.newEmail` | "Someone tried to change their account email to this address; no action needed" | none — generic greeting |
| `user.email_changed` | `data.oldEmail` | "Your account email was changed to <newEmail>; contact support if it wasn't you" | `firstName` (**added**) |
| `user.renamed` | — | no email | — |
| any other type (incl. topic `users.internal`) | — | ignored, offset committed | — |

Topic `orders.confirmed` (producer: order-service, unchanged snake_case format,
currently no `event-type` header): order confirmation email to
`data.customer_email`, same content as today's `order-confirmed` template.

### Confirmation links

`user.registration_requested` and `user.email_change_requested` carry
`encryptedToken` (base64 in JSON, NaCl anonymous sealed box) and `keyId`.
Notification-service opens it with `NOTIFICATION_PRIVATE_KEY` (same scheme as
`user-service/internal/nacl/opener.go`) and builds the link:

- registration: `{FRONTEND_URL}/confirm-registration?token={token}`
- email change: `{FRONTEND_URL}/confirm-email?token={token}`

Token is query-escaped. `FRONTEND_URL` is config; the paths are placeholders until
the frontend pages exist (user said "any for now"). The email also shows when the
link expires (`expiresAt`).

`keyId` handling: config has `NOTIFICATION_KEY_ID`; a message whose `keyId` does
not match is a permanent failure (logged, skipped). Single key only — no rotation
support.

## User-service change

In `../user-service`, add the account owner's full name to three domain events and
their wire format:

- `user.DuplicateRegistrationAttempted`, `user.EmailChangeRequested`,
  `user.EmailChanged` get `FullName FullName`, filled from `u.fullName` in
  `internal/user/user.go`.
- `internal/messaging/encoder.go`: their payloads get `firstName`, `lastName`.
- Update `encoder_test.go`, `internal/docs/asyncapi.yaml` (schemas + examples).
- Version stays `"1.0"` (additive field).

Not changed on purpose: `EmailChangeToTakenAddressAttempted` (the name would be
the requester's, sent to someone else) and `RegistrationRejected` (no email).

Notification-service must tolerate the new fields being absent (old messages
still in the topic): empty name → generic greeting.

## Style to follow (from user-service)

- Module `github.com/nexus-commerce/notification-service`, Go 1.27.2, stdlib
  `encoding/json/v2` for decoding like user-service's `messaging/decoder.go`.
- `cmd/consumer/main.go` (replaces `cmd/app`): `loadDotEnv()` via godotenv,
  `config.Load[consumerConfig]()` on `caarlos0/env/v6` with per-concern structs
  (`config.Kafka`, `config.Consumer`, `config.SendGrid`, `config.Notification`,
  `config.Frontend`), `signal.NotifyContext`, wire dependencies, `lifecycle.Run(ctx, consumers...)`.
- `internal/lifecycle`: copy of user-service's errgroup runner.
- `internal/kafka`: `Consumer` with `FetchMessage` → route by event type → commit,
  exponential retry, `ErrPermanent`, `Handler` interface `{EventType(); Handle(ctx, messaging.Message)}`,
  one handler type per event (like `reg_confirmed.go`). Sequential processing per
  consumer (drop today's goroutine-per-message, which commits nothing and loses
  ordering). One consumer per topic, both run under `lifecycle.Run`.
- `internal/messaging`: `Message`, header constant, envelope + typed decoders
  (`DecodeRegistrationRequested`, …) returning `ErrMalformedMessage`.
- `internal/usecase`: one use case per email (`SendRegistrationConfirmation`, …)
  depending on small consumer-side interfaces (`mailer`, `renderer`, `opener`).
- Adapters: `internal/sendgrid` (mailer), `internal/render` (embedded templates,
  kept and extended with new pages), `internal/nacl` (opener, copied from
  user-service).
- Errors wrapped with `%w` and context; no `fmt.Println` debugging; logging with
  `log.Printf` like user-service.
- Table-free, hand-written tests with fakes in `fakes_test.go`, in the same
  assertion style as user-service (`got = …, want …`).
- Dockerfile builds `./cmd/consumer` on `golang:1.27.2-alpine`; `.air.toml`,
  `.dockerignore`, `.env` (dev values, incl. the dev private key from
  user-service's `.env`), README with run/test commands.

## Error classification

- Malformed JSON / wrong data / bad token / key-id mismatch / template missing
  → permanent (skip, log).
- SendGrid network error, 5xx, 429, 401 or 403 → temporary (retry with backoff).
  Designer's call: 401/403 are credential/account failures, not message
  failures, so the message is retried rather than dropped.
- Other SendGrid 4xx → permanent.

## Constraints

- Delivery is at-least-once: a crash after send and before commit re-sends the
  email. Accepted, no dedup store.
- Consumer group: `KAFKA_CONSUMER_GROUP` default `notification-service`
  (differs from today's per-topic `notification-service-<topic>` groups).
  *Designer's call:* readers use `StartOffset: kafka.LastOffset`, so a group with
  no committed offset starts at the end of the topic instead of replaying all
  history and re-sending old welcome/order emails to real inboxes.
- *Designer's call:* confirmation emails whose `expiresAt` is already in the past
  are not sent (permanent outcome, logged). The two confirmation use cases take
  a clock (`func() time.Time`) so this is testable.

## Routing messages without an `event-type` header

order-service does not set the `event-type` header; `orders.confirmed` carries
its type only in the JSON body. *Designer's call:* `kafka.ConsumerOptions` gets a
`DefaultEventType string`; when a message has no `event-type` header the consumer
routes it as that type. The `orders.confirmed` consumer is created with
`DefaultEventType: "orders.confirmed"` and a handler whose `EventType()` returns
it. The `users` consumer leaves it empty, so header-less messages there are
ignored as in user-service. This is the one deliberate deviation from
user-service's consumer and must be covered by a test.

## Left to the builder

- Email copy and template markup for the new pages (English, reuse base layout).
- Exact package/file names inside the structure above.
- Sender address/name — keep current `MyEcom <contact@my-ecom-project.dynv6.net>`,
  as config with that default.
- Updating `nexus-infrastructure/docker-compose.yaml` env vars is out of scope
  but should be listed in the README.
