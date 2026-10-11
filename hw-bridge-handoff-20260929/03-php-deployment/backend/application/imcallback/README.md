# Tencent IM Callback Module

This module is isolated from the legacy `api/tencent/call_back` implementation.

## Environment

```text
IM_CALLBACK_TOKEN=<random-secret>
IM_SDK_APP_ID=<tencent-im-sdk-app-id>
IM_SDK_SECRET_KEY=<tencent-im-sdk-secret-key>
IM_CALLBACK_QUEUE_KEY=im:callback:queue
IM_CALLBACK_MAX_RETRIES=5
IM_CALLBACK_ADMIN_IDS=<comma-separated-admin-ids>
IM_CALLBACK_VERIFY_SIGNATURE=true
IM_BRIDGE_ENABLED=true
IM_BRIDGE_URL=http://bridge:8090
IM_BRIDGE_SHARED_SECRET=<same-secret-as-bridge>
```

ThinkPHP also accepts `PHP_IM_CALLBACK_TOKEN` and `PHP_IM_CALLBACK_SDK_APP_ID`.

## Endpoint

```text
POST /imcallback/tencent/receive?token=<random-secret>
Content-Type: application/json
```

Tencent appends `CallbackCommand`, `SdkAppid`, `ClientIP`, `OptPlatform`, and
`RequestId` as query parameters. The receiver merges these protocol fields
with the JSON body before validation and dispatch.

When Bridge delivery is enabled, after-send message events are stored in
`yy_im_bridge_delivery` and sent to `/internal/v1/events/tencent` with an
HMAC-SHA256 `X-Bridge-Signature` header. Bridge failures are retried by the
same callback worker and never change the Tencent callback response.

For local PHP built-in server testing:

```text
POST /index.php?s=/imcallback/tencent/receive&token=<random-secret>
```

## Database

Apply `database/001_create_im_callback_event.sql` and
`database/009_create_bridge_delivery.sql` before enabling Bridge delivery.

## First Live Test

Enable `C2C.CallbackAfterSendMsg` in the Tencent console, send one direct
message, and verify that exactly one row is written to `yy_im_callback_event`.
Sending the same fixture twice must increment `duplicate_count` without adding
a second row.

Isolated unit/integration tests:

```sh
IM_CALLBACK_TOKEN=<token> IM_SDK_APP_ID=<appid> \
php application/imcallback/tests/run.php
```

The runner uses a dedicated Redis queue key and `unit_*` data, then removes all
test events and projections in a `finally` block. It does not call the public
callback endpoint or modify legacy business tables.

Real Tencent end-to-end callback verification:

```sh
IM_SDK_APP_ID=<appid> IM_SDK_SECRET_KEY=<secret> IM_REST_REGION=sgp \
php application/imcallback/tests/tencent_e2e.php
```

This script calls Tencent REST APIs and waits for Tencent to deliver callbacks
through the configured public callback URL. It reuses the dedicated
`callback_e2e_a` and `callback_e2e_b` accounts and removes the temporary group
when the run finishes. Online-state callbacks still require a real SDK
login/logout client.

Linux smoke test:

```sh
CALLBACK_URL=http://SERVER_IP/imcallback/tencent/receive \
IM_CALLBACK_TOKEN=<random-secret> \
sh application/imcallback/tests/smoke.sh
```

An Nginx location example is available at
`deploy/nginx-location.conf.example`. Adapt the PHP-FPM upstream name to the
Docker Compose service name.

## Retention

Run `scripts/purge.php` daily from Docker Cron with `DB_*` environment
variables. The default retention period is 30 days.

## Worker

Run one compensation batch:

```sh
php application/imcallback/scripts/worker.php --once --limit=100
```

Run continuously under Supervisor, systemd, or a dedicated Docker service:

```sh
php application/imcallback/scripts/worker.php --limit=100
```

Redis accelerates delivery to the worker. Pending and failed database events
are scanned as a fallback, so a Redis outage does not lose callbacks.

## Admin

Apply `database/003_add_admin_menu.sql`, set `IM_CALLBACK_ADMIN_IDS`, then log
in again. The admin menu contains:

- IM callback logs and protected raw JSON details.
- Group, member, relation, profile, and online-state projections.

## Callback Rollout

Enable callbacks in batches and verify each batch before enabling the next:

1. Direct and group message callbacks.
2. Group lifecycle and member callbacks.
3. Friend, blacklist, and profile callbacks.
4. Online-state callbacks.

Robot, Chat push, message editing, message extensions, and clear-all-unread
callbacks are intentionally outside the current registry.
