# Local Development Changes

This file records local-only compatibility changes made while running
`www.demo.com` with the PHP built-in server. It is outside `wwwroot` and should
not be copied into the production image.

## Remove Before Production Packaging

Search marker:

```powershell
rg -n "TODO\(local-dev\)" F:\IM\wwwroot\www.demo.com
```

Local route compatibility changes:

- `public/admin/common/common.js`
  - Restore `WebHost` from `/index.php?s=/admin/` to `/admin/` after Nginx
    rewrite is configured.
- `public/admin/index.html`
  - Restore `&login_token=` to `?login_token=` after restoring `WebHost`.
- `public/admin/page/login.html`
  - Restore captcha and login URLs to the original `WebHost` routes.
- `public/admin/page/login/mobile.html`
  - Restore captcha query separators from `&t=` to `?t=`.

## Review Before Production Packaging

Tencent IM failure guards were added to prevent PHP 8 from throwing a 500
when the external Tencent endpoint returns `false`:

- `application/api/model/Tencent.php`
- `application/admin/controller/Group.php`

These guards are not required by the built-in server itself. They are safe
fallback behavior, but should be reviewed once Tencent IM credentials and
network access are available in Docker.

## Local Runtime Only

- Portable PHP lives under `F:\IM\.tools`.
- The Redis extension was enabled in `.tools/php-8.0.30/php.ini`.
- MySQL contains the local `local_admin` account.
- No database schema changes were made.

## Expected Docker Direction

- Serve `public` as the document root.
- Configure Nginx/Apache rewrite to `public/index.php`.
- Run PHP-FPM from the site context expected by relative paths.
- Install required PHP extensions, including Redis, MySQL, GD, mbstring,
  curl, fileinfo, and OpenSSL.
