# Notification-service Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild notification-service in user-service's style so it sends every email the current user-service events and order-service's `orders.confirmed` need.

**Architecture:** Kafka consumers (one per topic, run under an errgroup `lifecycle.Run`) route messages by `event-type` header to per-event handlers. A handler decodes the payload with `messaging` decoders into plain `event` structs and calls one `usecase` per email. Use cases depend on small interfaces implemented by adapters: `render` (embedded HTML templates), `sendgrid` (mailer), `nacl` (token opener).

**Tech Stack:** Go 1.27.2, `encoding/json/v2`, `github.com/segmentio/kafka-go`, `github.com/sendgrid/sendgrid-go`, `github.com/caarlos0/env/v6`, `github.com/joho/godotenv`, `golang.org/x/sync/errgroup`, `golang.org/x/crypto/nacl/box`, `html/template` + `embed`.

**Spec:** `docs/superpowers/specs/2026-11-08-notification-service-rewrite-design.md`

## Global Constraints

- Module path `github.com/nexus-commerce/notification-service`; `go 1.27.2`.
- Reference style is `../user-service`. When a step says "copy from user-service", copy the file and change only the import paths plus what the step names.
- Tests: plain `testing`, hand-written fakes, messages in the form `Func() = got, want want` / `Func() error: %v`. No testify.
- A plaintext confirmation token never appears in logs or error strings.
- Event types on topic `users`: `user.registration_requested`, `user.registered`, `user.duplicate_registration_attempted`, `user.email_change_requested`, `user.email_change_to_taken_address_attempted`, `user.email_changed`. Topic `orders.confirmed` uses type `orders.confirmed`.
- Kafka header with the event type: `event-type`.
- Confirmation links: `{FRONTEND_URL}/confirm-registration?token={token}` and `{FRONTEND_URL}/confirm-email?token={token}`, token query-escaped. Login link: `{FRONTEND_URL}/login`.
- Sender default: name `MyEcom`, address `contact@my-ecom-project.dynv6.net`.
- Readers use `StartOffset: kafka.LastOffset`; default consumer group `notification-service`.
- Empty first name → greeting `Hello,`; otherwise `Hi {FirstName},`.
- user-service is on branch `feature/email-change-confirmation` with a large uncommitted change set. **Do not commit in user-service**: leave Task 1's edits in its working tree and report them.
- notification-service: commit after each task on the current branch (create a feature branch first if executing on `main`).

## Review Focus

1. Message without `event-type` header on topic `orders.confirmed` → still handled as `orders.confirmed` (Task 5 test `TestRunUsesDefaultEventType`).
2. Old `users` messages without the new `firstName` fields → email sent with `Hello,` (Task 7 test `TestSendEmailChangedNoticeWithoutName`).
3. Event with empty recipient address → permanent skip, nothing sent, partition not blocked (Task 3 test `TestDecodersRejectMissingRecipient`).
4. Confirmation whose `expiresAt` already passed, or `keyId` mismatch, or token that cannot be opened → nothing sent, permanent skip, token not in error text (Task 6 tests).
5. SendGrid 429/5xx → retried; other 4xx → skipped (Task 8 test `TestSendClassifiesStatus`).

---

### Task 1: user-service — add names to three events

Work in `../user-service`. No commit (see Global Constraints).

**Files:**
- Modify: `../user-service/internal/user/event.go` (`DuplicateRegistrationAttempted`, `EmailChangeRequested`, `EmailChanged`)
- Modify: `../user-service/internal/user/user.go` (`RecordDuplicateRegistration`, `RequestEmailChange`, `ConfirmEmailChange`)
- Modify: `../user-service/internal/messaging/encoder.go` (`userDuplicateRegistrationAttempted`, `userEmailChangeRequested`, `userEmailChanged` + their convert funcs)
- Modify: `../user-service/internal/messaging/encoder_test.go`
- Modify: `../user-service/internal/docs/asyncapi.yaml` (the three payload schemas and examples)

**Interfaces:**
- Produces (wire format): the three payloads gain `"firstName"` and `"lastName"` strings, right after `userId`. Version stays `"1.0"`.

- [ ] **Step 1: Extend the three encoder tests.** In `TestEncodeUserDuplicateRegistrationAttempted`, `TestEncodeUserEmailChangeRequested`, `TestEncodeUserEmailChanged` build the event with `FullName: user.NewFullName(mustName(t, "Jane"), mustName(t, "Doe"))` (for `EmailChangeRequested` set it in `newUserEmailChangeRequested`), add `FirstName string \`json:"firstName"\`` and `LastName string \`json:"lastName"\`` to the payload `Data`, and assert `"Jane"`/`"Doe"`.
- [ ] **Step 2: Run** `go test ./internal/messaging/` in `../user-service`. Expected: compile FAIL (`unknown field FullName`).
- [ ] **Step 3: Implement.** Add `FullName FullName` to the three event structs; set `FullName: u.fullName` where `user.go` appends them; add `FirstName`/`LastName` fields to the three wire structs, filled from `e.FullName.FirstName().String()` / `LastName()`.
- [ ] **Step 4: Update `asyncapi.yaml`.** Add `firstName`/`lastName` (type string) to the three payload schemas' properties and `required`, and to their examples.
- [ ] **Step 5: Run** `go test ./... && go vet ./...` in `../user-service`. Expected: PASS.

---

### Task 2: Module, config, lifecycle, dev tooling

**Files:**
- Modify: `go.mod` (module path, `go 1.27.2`, deps), every `.go` file importing `notification-service/...`
- Create: `internal/config/config.go` (replaces the existing file), `internal/config/config_test.go`
- Create: `internal/lifecycle/lifecycle.go` (copy from user-service)
- Create: `.env` (local only), `.air.toml`, `.dockerignore`
- Modify: `.gitignore` (add `.env`, `tmp`, `.vscode`)

**Interfaces:**
- Produces:
  - `config.Load[T any]() (*T, error)` (copy from user-service)
  - `config.Kafka{Host string \`env:"KAFKA_HOST,required"\`; Port string \`env:"KAFKA_PORT" envDefault:"29092"\`}` with `func (k *Kafka) Addr() string`
  - `config.Consumer{GroupID string \`env:"KAFKA_CONSUMER_GROUP" envDefault:"notification-service"\`}`
  - `config.SendGrid{APIKey \`env:"SENDGRID_API_KEY,required"\`; FromName \`env:"MAIL_FROM_NAME" envDefault:"MyEcom"\`; FromAddress \`env:"MAIL_FROM_ADDRESS" envDefault:"contact@my-ecom-project.dynv6.net"\`}`
  - `config.Notification{PrivateKey \`env:"NOTIFICATION_PRIVATE_KEY,required"\`; KeyID \`env:"NOTIFICATION_KEY_ID,required"\`}`
  - `config.Frontend{URL string \`env:"FRONTEND_URL" envDefault:"http://localhost:5173"\`}`
  - `lifecycle.Runner` / `lifecycle.Run(ctx, runners ...Runner) error`

- [ ] **Step 1:** Set module path and `go 1.27.2` in `go.mod`, rewrite imports `notification-service/internal/...` → `github.com/nexus-commerce/notification-service/internal/...`. Run `go build ./...`. Expected: PASS (old code still builds; old `config.New` stays until Task 10, so keep it in the new file alongside `Load` for now).
- [ ] **Step 2: Write `TestLoadAppliesDefaults`**: `t.Setenv` `KAFKA_HOST=kafka`, `SENDGRID_API_KEY=k`, `NOTIFICATION_PRIVATE_KEY=p`, `NOTIFICATION_KEY_ID=dev`; `Load[struct{Kafka; Consumer; SendGrid; Notification; Frontend}]` → assert `Kafka.Addr() == "kafka:29092"`, `GroupID == "notification-service"`, `FromName == "MyEcom"`, `Frontend.URL == "http://localhost:5173"`. And `TestLoadRequiresKafkaHost`: unset → error.
- [ ] **Step 3:** Run `go test ./internal/config/`. Expected: FAIL (undefined `Load`).
- [ ] **Step 4:** Implement the config structs and `Load`; copy `lifecycle.go`. `go get github.com/caarlos0/env/v6 github.com/joho/godotenv golang.org/x/sync`.
- [ ] **Step 5:** Run `go test ./internal/config/ && go vet ./...`. Expected: PASS.
- [ ] **Step 6:** Create `.air.toml` (copy user-service's `.air.consumer.toml`, bin/cmd `./cmd/consumer`), `.dockerignore` (user-service's, minus relay/purge entries), `.gitignore` additions. Create `.env` with `KAFKA_HOST=localhost`, `KAFKA_PORT=9092`, `SENDGRID_API_KEY=`, `NOTIFICATION_KEY_ID` and `NOTIFICATION_PRIVATE_KEY` copied from `../user-service/.env`, `FRONTEND_URL=http://localhost:5173`. Confirm `git status` does not list `.env`.
- [ ] **Step 7: Commit** `chore: move to nexus-commerce module, add config loader and lifecycle`.

---

### Task 3: `event` structs and `messaging` decoders

**Files:**
- Create: `internal/event/event.go`
- Create: `internal/messaging/message.go`, `internal/messaging/decoder.go`, `internal/messaging/decoder_test.go`

**Interfaces:**
- Produces, package `event` (plain data, no methods):
  - `RegistrationRequested{UserID, Email, FirstName, LastName string; EncryptedToken []byte; KeyID string; ExpiresAt time.Time}`
  - `Registered{UserID, Email, FirstName, LastName string}`
  - `DuplicateRegistrationAttempted{UserID, Email, FirstName, LastName string}`
  - `EmailChangeRequested{UserID, NewEmail, FirstName, LastName string; EncryptedToken []byte; KeyID string; ExpiresAt time.Time}`
  - `EmailChangeToTakenAddressAttempted{UserID, NewEmail string}`
  - `EmailChanged{UserID, OldEmail, NewEmail, FirstName, LastName string}`
  - `OrderConfirmed` and `OrderItem`: same field names and types as today's `service.OrderData` / `service.OrderItemData`, except `OrderItem.ImageURL string`. Template field names must stay identical.
- Produces, package `messaging`:
  - `const HeaderEventType = "event-type"`, `TopicUsers = "users"`, `TopicOrdersConfirmed = "orders.confirmed"`, and type constants `TypeRegistrationRequested`, `TypeRegistered`, `TypeDuplicateRegistrationAttempted`, `TypeEmailChangeRequested`, `TypeEmailChangeToTakenAddressAttempted`, `TypeEmailChanged`, `TypeOrderConfirmed` with the values in Global Constraints.
  - `type Message struct{ Type, Topic, Key string; Value []byte }`
  - `var ErrMalformedMessage`
  - `DecodeRegistrationRequested([]byte) (event.RegistrationRequested, error)` and likewise `DecodeRegistered`, `DecodeDuplicateRegistrationAttempted`, `DecodeEmailChangeRequested`, `DecodeEmailChangeToTakenAddressAttempted`, `DecodeEmailChanged`, `DecodeOrderConfirmed`.

User decoders use user-service's envelope (`eventType`, `data`, camelCase data keys as in user-service `encoder.go` plus Task 1's `firstName`/`lastName`) via a generic `decodeData[T]` like user-service's `decoder.go`. `DecodeOrderConfirmed` uses the snake_case envelope (`event_type` must equal `orders.confirmed`, `data` with today's snake_case keys).

- [ ] **Step 1: Write tests.** `TestDecodeRegistrationRequested` (full payload with `"encryptedToken":"c2VhbGVk"` → `EncryptedToken == []byte("sealed")`, all fields equal), one happy-path test per other decoder, `TestDecodeEmailChangedWithoutName` (no name keys → empty `FirstName`, no error), `TestDecodeOrderConfirmed` (snake_case sample with one item → `CustomerEmail`, `OrderID`, `Items[0].ImageURL`). `TestDecodersRejectMalformed`: subtests `not json`, `other type`, `data not object` → `errors.Is(err, ErrMalformedMessage)`. `TestDecodersRejectMissingRecipient`: each decoder with its recipient field empty → `ErrMalformedMessage`. `TestDecodeConfirmationsRejectMissingExpiry`: registration/email-change requested with no `expiresAt` → `ErrMalformedMessage`.
- [ ] **Step 2:** Run `go test ./internal/messaging/`. Expected: FAIL (undefined).
- [ ] **Step 3:** Implement. Recipient = `email` / `newEmail` / `oldEmail` (for `EmailChanged`) / `customer_email`; also require non-empty `encryptedToken` for the two confirmation events.
- [ ] **Step 4:** Run `go test ./internal/messaging/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add event types and message decoders`.

---

### Task 4: `nacl` opener

**Files:**
- Create: `internal/nacl/opener.go`, `internal/nacl/opener_test.go`

**Interfaces:**
- Produces: `nacl.NewOpener(privateKey, keyID string) (*Opener, error)`; `(*Opener).Open(sealed []byte) ([]byte, error)`; `(*Opener).KeyID() string`; errors `ErrInvalidPrivateKey`, `ErrEmptyKeyID`, `ErrCannotOpen`.

Copy user-service's `opener.go`, add `keyID` (empty → `ErrEmptyKeyID`) and `KeyID()`. `keySize = 32` moves into this file.

- [ ] **Step 1: Write tests.** `TestOpenRoundTrip`: generate a pair with `box.GenerateKey(rand.Reader)`, seal `"token-123"` with `box.SealAnonymous`, `Open` → `"token-123"`. `TestOpenRejectsTampered` → `ErrCannotOpen`. `TestNewOpenerRejectsBadKey` (`"nope"`) → `ErrInvalidPrivateKey`. `TestNewOpenerRejectsEmptyKeyID` → `ErrEmptyKeyID`.
- [ ] **Step 2:** Run `go test ./internal/nacl/`. Expected: FAIL.
- [ ] **Step 3:** Implement; `go get golang.org/x/crypto`.
- [ ] **Step 4:** Run `go test ./internal/nacl/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add nacl token opener`.

---

### Task 5: Kafka consumer with default event type

**Files:**
- Create: `internal/kafka/consumer.go`, `internal/kafka/consumer_test.go` (copy both from user-service)

**Interfaces:**
- Consumes: `messaging.Message`, `messaging.HeaderEventType`.
- Produces: `kafka.ErrPermanent`; `kafka.Handler{EventType() string; Handle(ctx, messaging.Message) error}`; `kafka.ConsumerOptions{Addr, GroupID, Topic, DefaultEventType string}`; `kafka.NewConsumer(opts, handlers ...Handler) (*Consumer, error)`; `(*Consumer).Run(ctx) error`; `(*Consumer).Close() error`.

Deviations from the copy: reader config gets `StartOffset: kafka.LastOffset`; `toMessage(km, defaultType)` sets `Type = defaultType` when the header is absent; `Consumer` stores `defaultType`.

- [ ] **Step 1: Write tests** (beyond the copied ones, adjusting `toMessage` calls): `TestToMessageUsesDefaultEventType` (no header, default `"orders.confirmed"` → `Type == "orders.confirmed"`), `TestToMessageHeaderOverridesDefault`, and `TestRunUsesDefaultEventType`: `fakeReader` returning one header-less message then blocking until ctx cancel; consumer with `defaultType: "orders.confirmed"` and a `fakeHandler{typ: "orders.confirmed"}` → handler called once, message committed.
- [ ] **Step 2:** Run `go test ./internal/kafka/`. Expected: FAIL.
- [ ] **Step 3:** Implement the deviations.
- [ ] **Step 4:** Run `go test ./internal/kafka/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add kafka consumer with default event type`.

---

### Task 6: Use-case core and confirmation emails

**Files:**
- Create: `internal/usecase/usecase.go` (shared types, interfaces, errors, helpers), `internal/usecase/send_registration_confirmation.go`, `internal/usecase/send_email_change_confirmation.go`
- Test: `internal/usecase/fakes_test.go`, `internal/usecase/send_registration_confirmation_test.go`, `internal/usecase/send_email_change_confirmation_test.go` (package `usecase_test`)

**Interfaces:**
- Consumes: `event.RegistrationRequested`, `event.EmailChangeRequested`.
- Produces:
  - `type Email struct{ ToName, ToAddress, Subject, PlainText, HTML string }`
  - `type Mailer interface{ Send(ctx context.Context, e Email) error }`
  - `type Renderer interface{ Render(page string, data any) (string, error) }`
  - `type Opener interface{ Open(sealed []byte) ([]byte, error); KeyID() string }`
  - `var ErrUndeliverable` and, wrapping it: `ErrTokenExpired`, `ErrKeyMismatch`, `ErrInvalidToken`. Render failures are returned wrapped with `ErrUndeliverable` too. Adapters signal permanent failure by wrapping `ErrUndeliverable`.
  - `type Links struct{ base *url.URL }`; `NewLinks(frontendURL string) (Links, error)`; methods `ConfirmRegistration(token string) string`, `ConfirmEmail(token string) string`, `Login() string`.
  - Page name constants: `PageConfirmRegistration = "confirm-registration"`, `PageWelcome = "welcome"`, `PageDuplicateRegistration = "duplicate-registration"`, `PageConfirmEmail = "confirm-email"`, `PageEmailTaken = "email-taken"`, `PageEmailChanged = "email-changed"`, `PageOrderConfirmed = "order-confirmed"`.
  - `NewSendRegistrationConfirmation(r Renderer, m Mailer, o Opener, l Links, now func() time.Time) *SendRegistrationConfirmation`; `Execute(ctx, event.RegistrationRequested) error`. Renders `PageConfirmRegistration` with data `{FirstName, Link string; ExpiresAt time.Time}`; subject `Confirm your registration`; to `{FirstName LastName trimmed, Email}`.
  - `NewSendEmailChangeConfirmation(...)` same parameters; renders `PageConfirmEmail` with `{FirstName, Link, NewEmail string; ExpiresAt time.Time}`; subject `Confirm your new email address`; to `NewEmail`.

Order inside both: key ID check (`ErrKeyMismatch`) → expiry check `!now().Before(ExpiresAt)` (`ErrTokenExpired`) → open (`ErrInvalidToken`, original error not included) → render → send. Mailer errors are returned unchanged.

- [ ] **Step 1: Write fakes** in `fakes_test.go`: `fakeRenderer` (records `page`, `data`; returns `"<html>"+page`, or `err`), `fakeMailer` (records `[]usecase.Email`, optional `err`), `fakeOpener{keyID string; plain map[string]string}` (unknown input → error), fixed clock `func() time.Time { return time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC) }`.
- [ ] **Step 2: Write tests.** `TestSendRegistrationConfirmation`: token `"tok en/1"` → one email to `jane@example.com`, `ToName == "Jane Doe"`, subject as above, rendered page `confirm-registration`, `data.Link == "http://front.test/confirm-registration?token=tok+en%2F1"`. `TestSendRegistrationConfirmationSkipsExpired` (`ExpiresAt == now`) → `errors.Is(err, ErrTokenExpired)` and `errors.Is(err, ErrUndeliverable)`, no email. `…RejectsKeyMismatch`, `…RejectsUnopenableToken` (also assert `!strings.Contains(err.Error(), "tok")`). `…ReturnsMailerError` (mailer err passed through, not `ErrUndeliverable`). `…RenderFailureIsUndeliverable`. Mirror the happy-path, expired and key-mismatch tests for `SendEmailChangeConfirmation` (link `…/confirm-email?token=…`, recipient `new@example.com`). `TestNewLinksTrimsTrailingSlash`: `"http://front.test/"` → `Login() == "http://front.test/login"`. `TestNewLinksRejectsInvalidURL` (`"://"`).
- [ ] **Step 3:** Run `go test ./internal/usecase/`. Expected: FAIL.
- [ ] **Step 4:** Implement `usecase.go` and both use cases. A private helper `send(ctx, r, m, page string, data any, e Email) error` renders into `e.HTML` and sends; another `openToken(o, now, keyID string, sealed []byte, expiresAt time.Time) (string, error)`.
- [ ] **Step 5:** Run `go test ./internal/usecase/`. Expected: PASS.
- [ ] **Step 6: Commit** `feat: add confirmation email use cases`.

---

### Task 7: Notice, welcome and order use cases

**Files:**
- Create: `internal/usecase/send_welcome.go`, `send_duplicate_registration_notice.go`, `send_email_taken_notice.go`, `send_email_changed_notice.go`, `send_order_confirmation.go` and a `_test.go` for each.

**Interfaces:**
- Consumes: Task 6's `Email`, `Mailer`, `Renderer`, `Links`, page constants, `send` helper.
- Produces (each with `Execute(ctx, <event>) error`):

| Constructor | Event | Page | Data | Subject | To |
|---|---|---|---|---|---|
| `NewSendWelcome(r, m, l)` | `Registered` | `welcome` | `{FirstName, LoginURL}` | `Welcome to MyEcom` | Email |
| `NewSendDuplicateRegistrationNotice(r, m, l)` | `DuplicateRegistrationAttempted` | `duplicate-registration` | `{FirstName, LoginURL}` | `Someone tried to register with your email` | Email |
| `NewSendEmailTakenNotice(r, m)` | `EmailChangeToTakenAddressAttempted` | `email-taken` | `{Email}` | `Someone tried to use your email address` | NewEmail, empty name |
| `NewSendEmailChangedNotice(r, m)` | `EmailChanged` | `email-changed` | `{FirstName, NewEmail}` | `Your email address was changed` | OldEmail |
| `NewSendOrderConfirmation(r, m)` | `OrderConfirmed` | `order-confirmed` | the event itself | `Your Order Confirmation - #{OrderID}` | CustomerEmail, name `CustomerFirstName CustomerLastName` |

`PlainText` for every email: one sentence matching the subject (exact copy left to implementer; must be non-empty).

- [ ] **Step 1: Write tests.** One happy-path test per use case asserting page, recipient address, name, subject and the data fields above. `TestSendEmailChangedNoticeWithoutName`: empty names → `ToName == ""`, `data.FirstName == ""`, email still sent. `TestSendEmailTakenNoticeHasNoName`: event struct has no name; assert `ToName == ""`. `TestSendOrderConfirmationSubject`: `OrderID 42` → `Your Order Confirmation - #42`.
- [ ] **Step 2:** Run `go test ./internal/usecase/`. Expected: FAIL.
- [ ] **Step 3:** Implement the five use cases using `send`.
- [ ] **Step 4:** Run `go test ./internal/usecase/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add notice, welcome and order email use cases`.

---

### Task 8: `render` and `sendgrid` adapters

**Files:**
- Modify: `internal/render/render.go` (replace `NewTemplateCache` with `Renderer`); `internal/render/render_test.go`
- Rename: `templates/pages/user-registered.page.gohtml` → `welcome.page.gohtml` (use `.LoginURL`, greeting rule)
- Create: `templates/pages/confirm-registration.page.gohtml`, `confirm-email.page.gohtml`, `duplicate-registration.page.gohtml`, `email-taken.page.gohtml`, `email-changed.page.gohtml` (base layout; English copy per spec's Emails table; confirmation pages show the link as button and as text, and `ExpiresAt` formatted `Jan 2, 2006 15:04 MST`)
- Keep: `order-confirmed.page.gohtml` unchanged
- Create: `internal/sendgrid/mailer.go`, `internal/sendgrid/mailer_test.go`

**Interfaces:**
- Consumes: `usecase.Email`, `usecase.ErrUndeliverable`, page constants.
- Produces:
  - `render.New() (*Renderer, error)`; `(*Renderer).Render(page string, data any) (string, error)` — page `"welcome"` ↔ file `welcome.page.gohtml`; unknown page → error.
  - `sendgrid.New(apiKey, fromName, fromAddress string) *Mailer`; `(*Mailer).Send(ctx, usecase.Email) error`. Internally holds `client interface{ SendWithContext(context.Context, *mail.SGMailV3) (*rest.Response, error) }` so tests can fake it (unexported `newMailer(client, fromName, fromAddress)`).

Status rules: transport error → returned wrapped (retryable); `2xx` → nil; `429` or `>= 500` → plain error (retryable); other → `fmt.Errorf("%w: sendgrid status %d", usecase.ErrUndeliverable, code)`. Never put the response body into the error.

- [ ] **Step 1: Write tests.** `TestRenderEveryPage`: for each of the seven page constants render with representative data (structs from Task 6/7 data shapes; for `order-confirmed` an `event.OrderConfirmed` with one item) → no error, non-empty. `TestRenderGreeting`: `welcome` with `FirstName ""` contains `Hello,`; with `"Jane"` contains `Hi Jane,`. `TestRenderConfirmationShowsLink`: `confirm-registration` output contains the link. `TestRenderUnknownPage` → error. `TestSendClassifiesStatus`: subtests 202→nil, 429→err not `ErrUndeliverable`, 503→same, 400→`ErrUndeliverable`; plus transport error → not `ErrUndeliverable`. `TestSendBuildsMessage`: fake client captures `SGMailV3`; assert from name/address, subject, recipient.
- [ ] **Step 2:** Run `go test ./internal/render/ ./internal/sendgrid/`. Expected: FAIL.
- [ ] **Step 3:** Implement renderer, templates and mailer.
- [ ] **Step 4:** Run `go test ./internal/render/ ./internal/sendgrid/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add template renderer and sendgrid mailer`.

---

### Task 9: Kafka handlers

**Files:**
- Create: `internal/kafka/handlers.go`, `internal/kafka/handlers_test.go`

**Interfaces:**
- Consumes: messaging decoders and `Type…` constants, `usecase.ErrUndeliverable`, `messaging.ErrMalformedMessage`.
- Produces: generic `type EventHandler[T any] struct{ typ string; decode func([]byte) (T, error); uc interface{ Execute(context.Context, T) error } }` implementing `Handler`, and constructors taking the matching use case (as a consumer-side interface):
  `NewRegistrationRequested`, `NewRegistered`, `NewDuplicateRegistrationAttempted`, `NewEmailChangeRequested`, `NewEmailChangeToTakenAddressAttempted`, `NewEmailChanged`, `NewOrderConfirmed`.

`Handle`: decode error → `fmt.Errorf("%w: %w", ErrPermanent, err)`; use-case error matching `usecase.ErrUndeliverable` → same wrapping; anything else returned unchanged.

- [ ] **Step 1: Write tests** with a fake use case: `TestHandlerPassesDecodedEvent` (valid `user.registered` payload → fake got `Email == "jane@example.com"`), `TestHandlerMalformedIsPermanent`, `TestHandlerUndeliverableIsPermanent`, `TestHandlerTransientErrorPassesThrough` (not `ErrPermanent`), `TestHandlerEventTypes` (each constructor's `EventType()` equals its messaging constant).
- [ ] **Step 2:** Run `go test ./internal/kafka/`. Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run `go test ./internal/kafka/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add kafka event handlers`.

---

### Task 10: Wire `cmd/consumer`, remove old code, packaging

**Files:**
- Create: `cmd/consumer/main.go`
- Delete: `cmd/app/`, `internal/consumer/`, `internal/service/`, old `config.New`
- Modify: `Dockerfile`, `README.md`, `go.mod`/`go.sum` (`go mod tidy`)

**Interfaces:**
- Consumes: everything above.

`main.go` follows user-service's `cmd/consumer/main.go`: `loadDotEnv`, `config.Load[consumerConfig]` with `Kafka, Consumer, SendGrid, Notification, Frontend`, `signal.NotifyContext`, build `nacl.NewOpener`, `render.New`, `sendgrid.New`, `usecase.NewLinks`, use cases with `time.Now`. Two consumers: topic `messaging.TopicUsers` with the six user handlers; topic `messaging.TopicOrdersConfirmed` with `DefaultEventType: messaging.TypeOrderConfirmed` and the order handler; both with `cfg.Consumer.GroupID`; deferred `Close` with log on error; `return lifecycle.Run(ctx, users, orders)`.

- [ ] **Step 1:** Write `main.go`, delete old packages, `go mod tidy` (drop `cleanenv`).
- [ ] **Step 2:** Run `go build ./... && go vet ./... && go test ./...`. Expected: PASS, no reference to `cleanenv`, `internal/service`, `internal/consumer`.
- [ ] **Step 3:** Dockerfile: user-service's, `go build -o main ./cmd/consumer`, no `EXPOSE`. Run `docker build -t notification-service:dev .` if Docker is available. Expected: build succeeds (if Docker is unavailable, say so in the report).
- [ ] **Step 4:** README: purpose, emails table (from spec), env vars table (all vars from Task 2 with defaults/required), `go test ./...` / `go vet ./...`, `air` for dev, note that `nexus-infrastructure` docker-compose and `k8s/deployments/notification-service.yaml` still need `NOTIFICATION_PRIVATE_KEY`, `NOTIFICATION_KEY_ID`, `FRONTEND_URL`, `KAFKA_CONSUMER_GROUP`.
- [ ] **Step 5: Smoke run** (if a local Kafka is reachable at `.env`'s address): `go run ./cmd/consumer` → logs `Starting kafka consumer notification-service on topic users` and `… orders.confirmed`; Ctrl+C stops cleanly. Otherwise report that the smoke run was skipped.
- [ ] **Step 6: Commit** `refactor: wire new consumer and remove legacy service`.
