# notification-service

Kafka consumer that sends transactional emails (via SendGrid) in response to events from the user and order services. It has no HTTP API and no database.

It runs two consumers in one process, both in the consumer group `KAFKA_CONSUMER_GROUP`:

- topic `users`, six event types (see below);
- topic `orders.confirmed`, whose messages carry no `event-type` header and are treated as `orders.confirmed`.

Tokens in confirmation links are sealed by the producing service and opened here with `NOTIFICATION_PRIVATE_KEY` (NaCl box, selected by `NOTIFICATION_KEY_ID`).

## Emails

| Event                                           | Recipient        | Email                                              |
|-------------------------------------------------|------------------|----------------------------------------------------|
| `user.registration_requested`                   | `email`          | Confirm registration (link)                        |
| `user.registered`                               | `email`          | Welcome                                            |
| `user.duplicate_registration_attempted`         | `email`          | "Someone tried to register with your address"      |
| `user.email_change_requested`                   | `newEmail`       | Confirm new address (link)                         |
| `user.email_change_to_taken_address_attempted`  | `newEmail`       | Notice that the address is taken (no name)         |
| `user.email_changed`                            | `oldEmail`       | "Your email was changed"                           |
| `orders.confirmed`                              | `customer_email` | Order confirmation                                 |
| `user.registration_rejected`                    | -                | No email                                           |
| `user.renamed`                                  | -                | No email                                           |

## Configuration

Variables are read from the environment; a local `.env` file is loaded if present.

| Variable                   | Required | Default                            | Description                                   |
|----------------------------|----------|------------------------------------|-----------------------------------------------|
| `KAFKA_HOST`               | yes      | -                                  | Kafka broker host                             |
| `KAFKA_PORT`               | no       | `29092`                            | Kafka broker port                             |
| `KAFKA_CONSUMER_GROUP`     | no       | `notification-service`             | Consumer group id                             |
| `SENDGRID_API_KEY`         | yes      | -                                  | SendGrid API key                              |
| `MAIL_FROM_NAME`           | no       | `NexusEcommerce`                           | Sender display name                           |
| `MAIL_FROM_ADDRESS`        | yes      | -                                  | Sender address, on a SendGrid-verified domain |
| `NOTIFICATION_PRIVATE_KEY` | yes      | -                                  | Private key used to open sealed tokens        |
| `NOTIFICATION_KEY_ID`      | yes      | -                                  | Id of the key pair, must match the token's    |
| `FRONTEND_URL`             | no       | `http://localhost:5173`            | Base URL for links in emails                  |

## Development

```sh
go test ./...
go vet ./...
air            # rebuild and restart ./cmd/consumer on change (see .air.toml)
go run ./cmd/consumer
```

Docker image: `docker build -t notification-service:dev .`

## Infrastructure

The `nexus-infrastructure` docker-compose file and `k8s/deployments/notification-service.yaml` still need the new variables: `NOTIFICATION_PRIVATE_KEY`, `NOTIFICATION_KEY_ID`, `FRONTEND_URL` and `KAFKA_CONSUMER_GROUP`.
