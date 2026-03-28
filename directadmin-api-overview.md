# DirectAdmin API — Deep Overview

**Source:** https://docs.directadmin.com/developer/api/
**Live Swagger spec:** https://demo.directadmin.com:2222/static/swagger.json
**Interactive Swagger UI:** https://petstore.swagger.io/?url=https://demo.directadmin.com:2222/static/swagger.json
**Legacy endpoint changelog search:** https://www.directadmin.com/search_versions.php?help=no&versions=yes&query=CMD_API_
**Feature detail pages:** https://www.directadmin.com/features.php?id=NNN (replace NNN with feature ID)

---

## Overview

DirectAdmin exposes two distinct APIs that coexist on the same server:

| API | Prefix | Format | Status |
|-----|--------|--------|--------|
| **New JSON API** | `/api/...` | JSON, OpenAPI 2.0 | Preferred — more stable |
| **Legacy API** | `/CMD_API_...` or `/CMD_...` | URL-encoded (or JSON with `?json=yes`) | Use when new API lacks feature |

**Rule of thumb:** Always check the new API first. Fall back to the legacy API only when the feature is not yet ported.

---

## Authentication

### Basic HTTP Authentication (RFC 7617)

All endpoints use HTTP Basic Auth. Credentials are `username:password` Base64-encoded in the `Authorization` header. Most HTTP clients handle this natively.

```bash
curl --user "username:password" "https://server:2222/api/..."
```

Port is typically `2222` (the DirectAdmin management port).

### Login Keys (Recommended for Production)

Instead of using the main account password, generate a Login Key with restricted permissions and optional IP allowlists. This is the preferred approach for automation.

- Manage via new API: `GET/POST /api/login-keys/keys`
- Or via the DirectAdmin UI → Security → Login Keys
- When using a login key, authenticate as `username:login_key_value`

Login keys support:
- Command allowlists/denylists (`allowCommands`, `denyCommands`)
- Network restrictions (`allowNetworks`)
- Expiry (`hasExpiry`, `expires`, `autoRemove`)
- Login permission toggle (`allowLogin`)

**Temporary root credentials:** Run `da api-url` on the server to generate credentials valid 24 hours — useful for scripting without storing passwords.

### User Impersonation (Admin/Reseller)

Append `|username` to the authenticating username to act as that user. No need to create separate credentials for each managed account.

```
# Admin "admin1" acting as user "john"
Authorization: Basic base64("admin1|john:adminpassword")
```

```bash
curl --user "admin1|john:adminpassword" "https://server:2222/api/db-show/databases"
```

---

## New JSON API (`/api/...`)

### General Characteristics

- Base path: `/`
- All endpoints under `/api/`
- Request bodies: JSON
- Responses: JSON
- Full OpenAPI 2.0 spec available at `/static/swagger.json` on any DirectAdmin server
- 323 endpoints across 50 functional tags (as of current spec)

### Error Response Codes (Common)

| Code | Meaning |
|------|---------|
| 400 | Bad request / invalid parameters |
| 401 | Unauthenticated |
| 402 | Account suspended or license issue |
| 403 | Forbidden (insufficient permissions) |
| 404 | Resource not found |
| 409 | Conflict (e.g., resource already exists or process running) |
| 419 | Another operation in progress |
| 429 | Rate limited |
| 490–499 | Domain-specific business logic errors |
| 500 | Internal server error |
| 501 | Not implemented |

---

## New API Endpoints by Category

### Admin
- `GET /api/admin-usage` — Get admin's resource usage

### Backend Search
- `GET /api/search/resources` — Search user-owned resources (`?q=query`)
- `GET /api/search/users` — List accessible users (`?q=query&limit=N`)
- `GET /api/search/users-extended` — Search users by name or owned domain

### ClamAV
- `GET /api/clamav` — List active ClamAV scan processes
- `POST /api/clamav` — Start scan (`{path}`)
- `DELETE /api/clamav/{pid}` — Cancel scan by PID

### cPanel Import
- `POST /api/cpanel-import/check-remote` — Test SSH to remote cPanel server
- `GET /api/cpanel-import/tasks` — List import tasks
- `POST /api/cpanel-import/tasks/start` — Start import
- `GET /api/cpanel-import/tasks/{id}` — Get task status
- `DELETE /api/cpanel-import/tasks/{id}` — Delete pending task
- `GET /api/cpanel-import/tasks/{id}/log` — Get import log
- `GET /api/cpanel-import/tasks/{id}/log-sse` — Stream import log (SSE)

### CustomBuild (Server Software)
- `GET /api/custombuild/actions` — List available CB actions
- `GET /api/custombuild/software` — List available software
- `GET /api/custombuild/updates` — List available updates
- `GET /api/custombuild/state` — Get running state and update count
- `GET /api/custombuild/state/sse` — Stream state changes
- `POST /api/custombuild/run` — Run a custombuild action
- `POST /api/custombuild/kill` — Kill running custombuild
- `GET/PATCH /api/custombuild/options` — Get/update CB options
- `GET/PATCH /api/custombuild/options-v2` — Get/patch CB options (v2, key-value pairs)
- `GET /api/custombuild/options/validate` — Validate options
- `GET /api/custombuild/versions` — List default app versions
- `GET /api/custombuild/versions-custom` — List custom app versions
- `PUT /api/custombuild/versions-custom/{app}` — Set custom version
- `DELETE /api/custombuild/versions-custom/{app}` — Remove custom version
- `GET /api/custombuild/compile-scripts` — List compile script metadata
- `GET /api/custombuild/compile-scripts/{app}` — Get default compile script
- `GET/PUT/DELETE /api/custombuild/compile-scripts-custom/{app}` — Get/set/reset custom compile script
- `GET /api/custombuild/logs` — List CB log files
- `GET /api/custombuild/logs/{logname}/sse` — Stream CB log (SSE)
- `DELETE /api/custombuild/logs/{logname}` — Delete CB log
- `GET /api/custombuild/removals` — List commands to remove obsolete software

### Database
- `GET /api/db-show/databases` — List databases (`?no-size` to skip size calc)
- `GET /api/db-show/databases/{database}` — Get database metadata
- `GET /api/db-show/databases/{database}/users` — List database users
- `GET /api/db-show/users` — List all DB users
- `GET /api/db-show/users/{dbuser}` — Get DB user info
- `GET /api/db-show/users/{dbuser}/databases` — List DBs accessible by user
- `GET /api/db-show/info` — Database server info
- `POST /api/db-manage/create-db` — Create empty database
- `POST /api/db-manage/create-user` — Create DB user
- `POST /api/db-manage/create-db-with-user` — Create database + user together
- `DELETE /api/db-manage/databases/{database}` — Delete database (`?drop-orphan-users`)
- `DELETE /api/db-manage/users/{dbuser}` — Delete DB user
- `POST /api/db-manage/clone-db` — Clone database
- `POST /api/db-manage/clone-dbuser` — Clone DB user
- `GET /api/db-manage/databases/{database}/export` — Export DB (`?gzip`)
- `GET /api/db-manage/databases/{database}/export-definition` — Export DB + user definition
- `POST /api/db-manage/databases/{database}/import` — Import SQL file (multipart)
- `POST /api/db-manage/databases/{database}/import-definition` — Import DB + user definition
- `POST /api/db-manage/databases/{database}/check` — Check tables
- `POST /api/db-manage/databases/{database}/optimize` — Optimize tables
- `POST /api/db-manage/databases/{database}/repair` — Repair tables
- `POST /api/db-manage/databases/{database}/fix-definers` — Fix broken definers (views, events, routines, triggers)
- `POST /api/db-manage/users/{dbuser}/change-password` — Change DB user password
- `POST /api/db-manage/users/{dbuser}/change-hosts` — Set DB user host patterns (wildcards, localhost, IPs)
- `PUT /api/db-manage/users/{dbuser}/databases/{database}/change-privs` — Set DB privileges
- `POST /api/phpmyadmin-sso/account-access` — Create phpMyAdmin SSO URL (full account)
- `POST /api/phpmyadmin-sso/database-access/{database}` — Create phpMyAdmin SSO URL (single DB)
- `GET/PUT /api/server-settings/db-config` — Get/set DB server config
- `POST /api/server-settings/db-config-test` — Test DB config

### Database Monitor
- `GET /api/db-monitor/processes` — List MySQL/MariaDB process list
- `POST /api/db-monitor/processes/{id}/kill` — Kill DB thread

### DirectAdmin Config
- `GET /api/server-settings/directadmin-conf/active` — Get active (merged) DA config
- `GET /api/server-settings/directadmin-conf/default` — Get default DA config
- `GET /api/server-settings/directadmin-conf/local` — Get local overrides
- `PUT /api/server-settings/directadmin-conf/local` — Replace local config
- `PATCH /api/server-settings/directadmin-conf/local` — Patch local config

### Email
- `GET /api/email-logs` — Query email logs by time range, address, domain, state, type
- `GET /api/email-logs/user` — Same but scoped to current user
- `GET /api/email-logs-summary` — Aggregated email log summary
- `GET /api/email-config/mobileconfig` — Download Apple Mail config profile
- `GET/PUT /api/server-settings/email/config` — Server-level email settings
- `GET/PUT /api/server-settings/email/outbound-filter` — Outbound email blacklist

### Email Vacation
- `GET /api/emailvacation/{domain}` — List all vacation configs for domain
- `GET /api/emailvacation/{domain}/{user}` — Get user vacation config
- `PUT /api/emailvacation/{domain}/{user}` — Create/update vacation config
- `DELETE /api/emailvacation/{domain}/{user}` — Remove vacation config

### Exec
- `POST /api/execute` — Execute a shell command under the authenticated user's Unix privileges

### File Manager
- `GET /api/filemanager/list` — List directory contents
- `GET /api/filemanager/tree` — Directory tree
- `GET /api/filemanager/download` — Download file
- `GET /api/filemanager/download-archive` — Download file/dir as archive
- `GET /api/filemanager/disk-usage` — Get disk usage for path
- `GET /api/filemanager/search-files` — Search by filename
- `GET /api/filemanager/search-text` — Search by file content
- `GET /api/filemanager/list-archive` — List archive contents
- `GET /api/filemanager/trash` — List trashed items
- `POST /api/filemanager-actions/upload` — Upload file (multipart)
- `POST /api/filemanager-actions/mkdir` — Create directory
- `POST /api/filemanager-actions/copy` — Copy file/dir
- `POST /api/filemanager-actions/copy-to` — Copy multiple items to directory
- `POST /api/filemanager-actions/move` — Move file/dir
- `POST /api/filemanager-actions/move-to` — Move multiple items to directory
- `POST /api/filemanager-actions/remove` — Delete files/dirs
- `POST /api/filemanager-actions/chmod` — Change permissions
- `POST /api/filemanager-actions/create-archive` — Archive files
- `POST /api/filemanager-actions/extract-archive` — Extract archive
- `DELETE /api/filemanager-actions/trash` — Empty trash
- `DELETE /api/filemanager-actions/trash/{id}` — Delete single trashed item
- `POST /api/filemanager-actions/trash/{id}/restore` — Restore trashed item

### Git
- `GET /api/git/domain/{domain}` — List repositories under a domain
- `POST /api/git/domain/{domain}` — Init or clone a repository
- `GET /api/git/uuid/{uuid}` — Get repository info
- `PUT /api/git/uuid/{uuid}` — Update repository settings
- `DELETE /api/git/uuid/{uuid}` — Remove repository
- `GET /api/git/uuid/{uuid}/branch/{branch}` — Branch commit log
- `GET /api/git/uuid/{uuid}/commit/{commit}` — Commit info
- `POST /api/git/uuid/{uuid}/fetch` — Fetch from remote
- `POST /api/git/uuid/{uuid}/deploy` — Deploy to working tree
- `POST /api/git/user/{username}/uuid/{uuid}/webhook` — Webhook: fetch + deploy

### Hostname
- `POST /api/server-settings/change-hostname` — Change server hostname

### IMAP Sync (imapsync)
- `POST /api/imapsync/import` — Import email from external IMAP server
- `POST /api/imapsync/export` — Export email to external IMAP server
- `GET /api/imapsync/migrations` — List running migrations
- `DELETE /api/imapsync/migrations/{id}` — Cancel migration

### Info / System Info
- `GET /api/info` — Basic server info (no auth required)
- `GET /api/system-info/cpu` — CPU usage
- `GET /api/system-info/memory` — Memory usage
- `GET /api/system-info/load` — System load
- `GET /api/system-info/uptime` — Uptime
- `GET /api/system-info/fs` — Filesystem usage
- `GET /api/system-info/services` — Service status overview

### Licensing
- `GET /api/license` — License info (admin only)
- `GET /api/license/proof` — Session-based license proof
- `POST /api/license/update-key` — Change license key

### Login Keys (API)
- `GET /api/login-keys/commands` — List all commands available for login key scoping
- `GET /api/login-keys/keys` — List login keys
- `POST /api/login-keys/keys` — Create login key
- `GET /api/login-keys/keys/{id}` — Get key details
- `PATCH /api/login-keys/keys/{id}` — Update key
- `DELETE /api/login-keys/keys/{id}` — Delete key
- `GET /api/login-keys/keys/{id}/history` — Key usage history
- `GET /api/login-keys/urls` — List login URLs
- `POST /api/login-keys/urls` — Create login URL (OTP-based)
- `DELETE /api/login-keys/urls/{id}` — Delete login URL

### Maintenance
- `GET /api/maintenance` — List maintenance tasks
- `POST /api/maintenance/{task}/check` — Run task checks
- `POST /api/maintenance/{task}/fix` — Auto-fix task

### Messages (Internal Messaging)
- `GET /api/messages` — Get messages list
- `GET /api/messages/list` — List messages with search
- `GET /api/messages/id/{id}` — Read message (marks as read)
- `POST /api/messages/read` — Mark messages as read (array of IDs)
- `POST /api/messages/delete` — Delete messages (array of IDs)

### Migrate
- `POST /api/change-user-creator` — Move user between resellers
- `POST /api/convert-user-to-reseller` — Promote user to reseller (admin only)
- `POST /api/convert-reseller-to-user` — Demote reseller to user (admin only)

### Misc
- `POST /api/restart` — Restart DirectAdmin process

### ModSecurity
- `GET /api/modsecurity-audit-log/summary` — Parse modsecurity audit log
- `GET /api/modsecurity-audit-log/entry` — Get raw log entry by reference
- `GET /api/modsecurity/all-configs` — ModSec config for all domains (admin only)
- `GET /api/modsecurity/user-configs` — ModSec config for user-owned domains
- `GET /api/modsecurity/configs/{hostname}` — Config for single hostname
- `PUT /api/modsecurity/configs/{hostname}` — Replace config for hostname
- `DELETE /api/modsecurity/configs/{hostname}` — Revert config for hostname to default
- `GET /api/modsecurity/global-config` — Global ModSec config (admin only)
- `PUT /api/modsecurity/global-config` — Replace global config (admin only)

### Password
- `POST /api/change-password` — Change Unix/mail/FTP password (`{currentPassword, newPassword}`)

### Plugin Manager
- `GET /api/plugin-manager/plugins` — List plugins and status
- `POST /api/plugin-manager/plugins/install-file` — Install from uploaded file
- `POST /api/plugin-manager/plugins/install-url` — Install from URL
- `POST /api/plugin-manager/plugins/{id}/install` — Install plugin
- `POST /api/plugin-manager/plugins/{id}/uninstall` — Uninstall plugin
- `POST /api/plugin-manager/plugins/{id}/activate` — Activate plugin
- `POST /api/plugin-manager/plugins/{id}/deactivate` — Deactivate plugin
- `POST /api/plugin-manager/plugins/{id}/update` — Update plugin
- `POST /api/plugin-manager/plugins/{id}/delete` — Delete plugin

### Plugins
- `GET /api/plugins/list` — List installed plugins (user-facing)

### Profile
- `GET /api/profile/settings` — Get user profile settings
- `PATCH /api/profile/settings` — Update profile settings

### Redis
- `GET /api/redis/status` — Redis status
- `POST /api/redis/enable` — Start Redis
- `POST /api/redis/disable` — Stop Redis

### Resellers
- `GET /api/resellers/{username}/config` — Get reseller config
- `GET /api/resellers/{username}/usage` — Get reseller + child users' usage

### Session Control
- `POST /api/login` — Create session (`{username, password, otp, redirectURL, logoutURL}`)
- `POST /api/login/url` — Login via one-time password (`?key=...`)
- `POST /api/logout` — Destroy current session
- `POST /api/session/login-as/switch` — Impersonate another account (`{username}`)
- `POST /api/session/login-as/return` — Return from impersonated session

### Sessions
- `GET /api/session` — Current session info
- `GET /api/session/state` — Server state
- `GET /api/session/user-config` — Current user's config
- `GET /api/session/user-usage` — Current user's usage
- `GET /api/session/reseller-config` — Reseller config (if applicable)
- `POST /api/session/switch-active-domain` — Change active domain
- `GET /api/sessions` — List all active sessions for user
- `POST /api/sessions/destroy-all-other` — Kill all other sessions
- `POST /api/sessions/destroy/{public_id}` — Kill specific session

### Skin Customization
- `GET /api/session/skin-customization/{skin}` — List active customizations
- `GET/PUT/DELETE/PATCH /api/skin-customization/{skin}/menu` — Menu customizations
- `GET /api/skin-customization/{skin}/local` — List your customization files
- `POST /api/skin-customization/{skin}/local` — Upload files
- `DELETE /api/skin-customization/{skin}/local` — Delete all custom files
- `GET/PUT/DELETE /api/skin-customization/{skin}/local/{filename}` — Single file CRUD
- Custom images (favicon, logo, logo2, symbol, symbol2): `POST/DELETE /api/skin-customization/{skin}/images/{type}`

### Skin Options
- `GET/PUT/DELETE/PATCH /api/skin-options/{skin}` — User skin options
- `GET/PUT/DELETE/PATCH /api/skin-global-options/{skin}` — Server-wide skin options (admin)

### Skin Translations
- `GET /api/skin-translation/{skin}/{lang}` — Merged skin translations in JSON

### System Packages / Updates
- `GET /api/system-packages/updates` — Available OS package upgrades
- `POST /api/system-packages/update-test` — Simulate upgrade
- `POST /api/system-packages/update-run` — Run upgrade
- `GET /api/system-packages/history` — Package manager history
- `GET /api/system-packages/history/{id}/log` — History entry log
- `GET /api/system-packages/history/{id}/sse` — Stream history log

### System Services
- `GET /api/system-services/list` — List all services
- `GET /api/system-services/service/{service}` — Get service properties
- `GET /api/system-services/service/{service}/log` — Service logs (with filters: cursor, from, to, level, limit)
- `POST /api/system-services-actions/service/{service}/start` — Start service
- `POST /api/system-services-actions/service/{service}/stop` — Stop service
- `POST /api/system-services-actions/service/{service}/restart` — Restart service
- `POST /api/system-services-actions/service/{service}/reload` — Reload service
- `PUT /api/system-services-actions/service/{service}/watchdog` — Toggle watchdog (`{enable}`)

### TLS / SSL
- `GET /api/domain-tls/{domain}/certs` — List domain TLS certificates
- `GET /api/domain-tls/{domain}/certs/{id}/files` — Retrieve certificate files
- `PUT /api/domain-tls/{domain}/certs/{id}/files` — Replace certificate files (`?force`)
- `DELETE /api/domain-tls/{domain}/certs/{id}` — Delete certificate
- `GET/PUT /api/domain-tls/{domain}/acme-config` — Domain ACME config
- `GET /api/server-tls/certificate` — Server TLS certificate info
- `GET /api/server-tls/files` — Server certificate files
- `PUT /api/server-tls/files` — Replace server certificate (`?force`)
- `GET /api/server-tls/status` — Certificate status
- `POST /api/server-tls/enable` — Enable SSL for main server
- `POST /api/server-tls/obtain` — Force-obtain TLS certificate
- `GET/PUT /api/server-tls/acme-config` — Server ACME config

### Tickets
- `GET /api/ticket-requests` — User's submitted ticket requests
- `GET /api/tickets` — Received tickets (admin/reseller only)

### Timezone
- `GET /api/server-settings/timezone/current` — Current server timezone
- `GET /api/server-settings/timezone/list` — Available timezones
- `POST /api/server-settings/timezone/set` — Set timezone (`{tz}`)

### Two-Factor Authentication
- `POST /api/multi-factor-auth/otp/generate-secret` — Generate new TOTP secret
- `POST /api/multi-factor-auth/enable` — Enable 2FA (`{otp}`)
- `POST /api/multi-factor-auth/disable` — Disable 2FA
- `GET /api/multi-factor-auth/recovery-codes` — View recovery codes
- `POST /api/multi-factor-auth/recovery-codes` — Regenerate recovery codes
- `PATCH /api/multi-factor-auth/settings` — Update 2FA settings (`{notifyAllFailed}`)

### Users
- `GET /api/users/{username}/config` — Get user config (negative `Lim` values = unlimited)
- `GET /api/users/{username}/usage` — Usage and limits
- `GET /api/users/{username}/login-history` — Login history
- `GET /api/login-history` — Current user's own login history

### Versioning (DirectAdmin Updates)
- `GET /api/version` — Version info (admin only)
- `PATCH /api/version` — Change update channel (admin only)
- `POST /api/version/update` — Update DirectAdmin (admin only)

### Web Protect (HTTP Basic Auth on Directories)
- `GET /api/web-protect/list` — List protected directories
- `POST /api/web-protect/add-dir` — Protect a directory
- `GET /api/web-protect/dirs/{id}` — Get protected dir details
- `DELETE /api/web-protect/dirs/{id}` — Remove protection
- `POST /api/web-protect/dirs/{id}/update-realm` — Update realm name
- `POST /api/web-protect/dirs/{id}/update-user` — Add/update user
- `POST /api/web-protect/dirs/{id}/delete-users` — Remove users

### Widgets
- `GET /api/widgets/list` — List available dashboard widgets

### WordPress
- `GET /api/wordpress/locations` — List known WordPress installations (`?domain=filter`)
- `POST /api/wordpress/install` — Full install with existing database
- `POST /api/wordpress/install-quick` — Quick install (creates DB automatically)
- `DELETE /api/wordpress/locations/{location_id}` — Remove WP (files + DB tables; DB user kept)
- `GET /api/wordpress/locations/{location_id}/wordpress` — WP install info
- `GET/PUT /api/wordpress/locations/{location_id}/config` — DB config
- `PUT /api/wordpress/locations/{location_id}/config/auto-update` — Toggle auto-update
- `GET /api/wordpress/locations/{location_id}/options` — WP options table
- `PATCH /api/wordpress/locations/{location_id}/options` — Update WP options
- `GET /api/wordpress/locations/{location_id}/users` — WP user accounts
- `POST /api/wordpress/locations/{location_id}/users/{user_id}/change-password` — Change WP user password
- `POST /api/wordpress/locations/{location_id}/users/{user_id}/sso-login` — Generate magic login URL

### Xterm (Web Terminal)
- `GET /api/terminal` — WebSocket endpoint for terminal session (`?cols=80&rows=24`)

### Lost Password
- `POST /api/lost-password/request` — Request password reset (`{username}`)
- `POST /api/lost-password/confirm` — Confirm reset with code (`{code}`)

### Security.txt
- `GET /api/security-txt/status` — Check security.txt presence/status

---

## Legacy API (`/CMD_API_...`)

### General Characteristics

- Endpoint prefix: `/CMD_API_` (machine-readable) or `/CMD_` (human-readable HTML)
- Default response: URL-encoded key-value string
- **JSON mode:** Append `?json=yes` to any legacy endpoint to receive JSON instead
- POST bodies: URL-encoded form data OR JSON (JSON support was added later)
- Checkbox fields: pass the field name with value `ON` to enable; **omit entirely** to disable (do not send `OFF`)
- Success: `error=0&text=...&details=...`
- Failure: `error=1&text=...&details=...`
- Lists: returned as `list[]=value1&list[]=value2` (URL-encoded) or JSON array

### Account Management

#### Create Admin
```
CMD_API_ACCOUNT_ADMIN  POST
  action=create, username, email, passwd, passwd2, notify=yes|no
  passwd_is_crypted=1  (optional — use pre-hashed password)
```

#### Create Reseller (package-based)
```
CMD_API_ACCOUNT_RESELLER  POST
  action=create, add=Submit
  username, email, passwd, passwd2, domain
  package=<reseller_package_name>
  ip=shared|sharedreseller|assign
  notify=yes|no
```

#### Create Reseller (custom, no package)
```
CMD_API_ACCOUNT_RESELLER  POST
  action=create, add=Submit
  username, email, passwd, passwd2, domain
  bandwidth=<MB>, ubandwidth=ON (unlimited)
  quota=<MB>, uquota=ON
  vdomains=<N>, uvdomains=ON
  nsubdomains=<N>, unsubdomains=ON
  ips=<N>
  nemails=<N>, unemails=ON
  nemailf=<N>, unemailf=ON
  nemailml=<N>, unemailml=ON
  nemailr=<N>, unemailr=ON
  mysql=<N>, umysql=ON
  domainptr=<N>, udomainptr=ON
  ftp=<N>, uftp=ON
  aftp=ON|OFF, php=ON|OFF, cgi=ON|OFF
  ssl=ON|OFF, ssh=ON|OFF, userssh=ON|OFF
  dnscontrol=ON|OFF
  dns=OFF|TWO|THREE
  serverip=ON|OFF
  ip=shared|assign
  notify=yes|no
```

#### Create User (package-based)
```
CMD_API_ACCOUNT_USER  POST
  action=create, add=Submit
  username, email, passwd, passwd2, domain
  package=<user_package_name>
  ip=<available_ip>
  notify=yes|no
```

#### Create User (custom, no package)
```
CMD_API_ACCOUNT_USER  POST
  action=create
  (all fields from reseller custom creation, plus:)
  inode=<N>, uinode=ON
  spam=ON|OFF, cron=ON|OFF, catchall=ON|OFF
  sysinfo=ON|OFF, login_keys=ON|OFF
  suspend_at_limit=ON|OFF
  skin=evolution|enhanced|...
  language=en|...
```

#### Delete Any Account Type
```
CMD_API_SELECT_USERS  POST
  confirmed=Confirm, delete=yes
  select0=username, select1=username2, ...
```

#### Suspend / Unsuspend Any Account
```
CMD_API_SELECT_USERS  POST
  location=CMD_ADMIN_SHOW|CMD_RESELLER_SHOW|CMD_SELECT_USERS
  suspend=Suspend|Unsuspend
  select0=username, select1=username2, ...
```

#### Modify User Settings
```
CMD_API_MODIFY_USER  POST
  action=customize, user=<username>
  (same resource fields as custom user creation)
  ns1=<nameserver1>, ns2=<nameserver2>
```

#### List Users
```
CMD_API_SHOW_USERS        GET  — users for calling reseller (or ?reseller=X)
CMD_API_SHOW_RESELLERS    GET  — all resellers (admin only)
CMD_API_SHOW_ADMINS       GET  — all admins (admin only)
CMD_API_SHOW_ALL_USERS    GET  — all users on server (admin only)
CMD_API_USER_EXISTS       GET  — check if user exists (returns exists=0|1)
```

### Server Information

#### Admin Statistics
```
CMD_API_ADMIN_STATS  GET
Returns: RX, TX, bandwidth, quota, disk1..diskN (colon-separated),
         domainptr, ftp, mysql, nemailf, nemailml, nemailr, nemails,
         nresellers, nsubdomains, nusers, vdomains
```

#### User Usage
```
CMD_API_SHOW_USER_USAGE  GET
  user=<username>
Returns: bandwidth, quota, domainptr, ftp, mysql, nemailf, nemailml,
         nemailr, nemails, nsubdomains, vdomains
```

#### User Config / Limits
```
CMD_API_SHOW_USER_CONFIG  GET
  user=<username>
Returns (40+ fields including): account, aftp, bandwidth, catchall,
  cgi, creator, cron, date_created, dnscontrol, domain, email, ftp,
  ip, language, login_keys, mysql, nemailf, nemailml, nemailr, nemails,
  ns1, ns2, nsubdomains, package, quota, skin, spam, ssh, ssl,
  suspend_at_limit, suspended, sysinfo, username, usertype, vdomains
  (negative values or "unlimited" = no limit)
```

#### User Domains Info
```
CMD_API_SHOW_USER_DOMAINS  GET
  user=<username>
Returns: one entry per domain:
  domain.com=bw_used:bw_limit:disk_usage:log_usage:subdomains:suspended:quota:ssl:cgi:php
```

### Reseller IPs
```
CMD_API_SHOW_RESELLER_IPS  GET|POST
  (ip=x.x.x.x)  — optional, for single IP info
Returns list[] of IPs, or single IP info: ns, reseller, status, value
```

### Packages
```
CMD_API_PACKAGES_RESELLER  GET         — list reseller packages
CMD_API_PACKAGES_RESELLER  GET ?package=<name>  — single package details
CMD_API_PACKAGES_USER      GET         — list user packages
CMD_API_PACKAGES_USER      GET ?package=<name>  — single package details
```

Package fields (reseller): aftp, cgi, dns, dnscontrol, bandwidth, domainptr, ftp, ips, mysql, nemailf, nemailml, nemailr, nemails, nsubdomains, quota, serverip, ssh, userssh, ssl, vdomains

Package fields (user): aftp, cgi, dnscontrol, bandwidth, domainptr, ftp, mysql, name, nemailf, nemailml, nemailr, nemails, nsubdomains, quota, skin, ssh, ssl, vdomains, suspend_at_limit

### Sessions
```
CMD_API_GET_SESSION  GET|POST
  ip=<client_ip>, session_id=<session_cookie_value>
Returns: error, username, usertype, password (base64 encoded)
```

### Domain Operations (User Level)
```
CMD_API_SHOW_DOMAINS  GET|POST        — list user domains (returns list[])

CMD_API_DOMAIN  POST  action=create
  domain, bandwidth=<MB>|ubandwidth=unlimited,
  quota=<MB>|uquota=unlimited, ssl=ON|OFF, cgi=ON|OFF, php=ON|OFF

CMD_API_SUBDOMAINS  GET|POST  action=list&domain=domain.com
CMD_API_SUBDOMAINS  GET|POST  action=create&domain=domain.com&subdomain=sub
CMD_API_SUBDOMAINS  GET|POST  action=delete&domain=domain.com&select0=sub&contents=yes|no
```

### Email Operations (User Level)
```
CMD_API_POP  GET|POST  action=list&domain=domain.com
CMD_API_POP  GET|POST  action=create&domain=X&user=bob&passwd=P&passwd2=P&quota=<MB>&limit=N
CMD_API_POP  GET|POST  action=delete&domain=X&user=bob

CMD_CHANGE_EMAIL_PASSWORD  GET|POST   — change virtual email password WITHOUT DA auth
  email=user@domain.com, oldpassword=X, password1=X, password2=X
  api=yes  (returns url-encoded response instead of HTML)
  redirect=<url>  (optional redirect on success)
```

### Account Info Changes
```
CMD_API_CHANGE_INFO  POST
  evalue=<email>, domain=<any_user_domain>, email=Save

CMD_API_TICKET  POST
  email=<addr>, ON=yes|no, save=Save
```

### Crypted Password Support
When creating accounts via CMD_API_ACCOUNT_USER/RESELLER/ADMIN:
```
passwd=$hash$cryptedvalue
passwd_is_crypted=1
```

### Auto-Login from External Sites
```html
<form action="https://server:2222/CMD_LOGIN" method="POST" name="form">
  <input type="hidden" name="referer" value="/">
  <input type="hidden" name="FAIL_URL" value="http://yourdomain.com/fail.html">
  <input type="hidden" name="LOGOUT_URL" value="http://yourdomain.com/logout.html">
  <input type="hidden" name="username" value="username">
  <input type="hidden" name="password" value="login_key_value">
</form>
<script>document.form.submit();</script>
```

---

## Code Examples

### List Databases — New API (Bash/Python/PHP)

```bash
curl -s --user "user:pass" "https://server:2222/api/db-show/databases"
# With jq to extract names:
curl -s --user "user:pass" "https://server:2222/api/db-show/databases" | jq .[].database -r
```

```python
import requests
response = requests.get("https://server:2222/api/db-show/databases", auth=("user", "pass"))
for db in response.json():
    print(db['database'])
```

```php
$curl = curl_init();
curl_setopt($curl, CURLOPT_URL, "https://server:2222/api/db-show/databases");
curl_setopt($curl, CURLOPT_USERPWD, "user:pass");
curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
$data = json_decode(curl_exec($curl), true);
```

### List All Users — Legacy API with JSON mode

```bash
curl -s --user "admin:pass" "https://server:2222/CMD_API_SHOW_ALL_USERS?json=yes" | jq .[]
```

```python
import requests
response = requests.get("https://server:2222/CMD_API_SHOW_ALL_USERS?json=yes", auth=("admin", "pass"))
for user in response.json():
    print(user)
```

### Install WordPress — New API

```python
import requests
payload = {
    "adminEmail": "admin@example.com",
    "adminName": "Admin",
    "adminPass": "SecurePass123!",
    "filePath": "domains/example.com/public_html",
    "title": "My Site"
}
response = requests.post("https://server:2222/api/wordpress/install-quick",
                         auth=("user", "pass"), json=payload)
```

### Create User — Legacy API

```python
import requests
payload = {
    "action": "create", "add": "yes", "json": "yes",
    "username": "newuser", "email": "user@example.com",
    "passwd": "Password123!", "passwd2": "Password123!",
    "domain": "newuser.com", "package": "default",
    "ip": "192.0.2.1", "notify": "yes"
}
response = requests.post("https://server:2222/CMD_ACCOUNT_USER?json=yes",
                         auth=("admin", "pass"), json=payload)
```

---

## Discovering Undocumented Endpoints

### The Versions / Changelog System (Primary Method for Legacy CMD_API_)

The official docs explicitly note that most `CMD_API_` calls are **not documented in the legacy API page** — they are only listed in the DirectAdmin changelog/versions system:

**Search URL:** `https://www.directadmin.com/search_versions.php?help=no&versions=yes&query=CMD_API_`

**How it works:**
- It is a **changelog search**, not an API reference. It returns a list of changelog entries whose titles mention the query string.
- Each result has a title describing the feature and a numeric ID.
- Results are sorted newest-first (highest ID = most recent).
- Entry types are `feature`, `bugfix`, or `behavior change`.
- **Version numbers are NOT shown on the search results page.** To see which DA version introduced a command, click through to the individual feature page: `https://www.directadmin.com/features.php?id=NNN`
- **No parameter documentation** is shown anywhere in this system — just the changelog description.

**How to use it to find a command:**
1. Search for a keyword: `?query=CMD_API_DNS`, `?query=CMD_API_EMAIL`, `?query=backup`, etc.
2. Read the changelog titles to find the relevant entry.
3. Visit `features.php?id=NNN` to see which DA version added it and any notes in the entry.
4. Use the command name from the title, then either test it against a live server or inspect it via browser DevTools.

**Useful search queries:**
| Query | Finds |
|-------|-------|
| `CMD_API_` | All 162+ legacy API changelog entries |
| `CMD_API_DNS` | DNS-related API commands |
| `CMD_API_EMAIL` | Email API commands |
| `CMD_API_FTP` | FTP API commands |
| `CMD_API_DB` | Database-related API commands |
| `backup` | Backup API entries |
| `json` | Entries about JSON support additions |

### Complete CMD_API_ Inventory (from changelog search)

All known legacy API commands extracted from the versions system as of DA 1.697. Many have no further parameter documentation — use browser DevTools or server debug mode to discover their parameters.

**Account & User Management**
- `CMD_API_ACCOUNT_ADMIN` — Create admin account
- `CMD_API_ACCOUNT_RESELLER` — Create reseller account
- `CMD_API_ACCOUNT_USER` / `CMD_ACCOUNT_USER` — Create user account
- `CMD_API_SELECT_USERS` — Delete or suspend/unsuspend any account type
- `CMD_API_MODIFY_USER` — Modify user resource limits and settings
- `CMD_API_MODIFY_RESELLER` — Modify reseller settings (also `action=single` for bandwidth/quota/inode)
- `CMD_API_SHOW_USERS` — List users for a reseller
- `CMD_API_SHOW_RESELLERS` — List all resellers (admin)
- `CMD_API_SHOW_ADMINS` — List all admins (admin)
- `CMD_API_SHOW_ALL_USERS` — List all users on server (admin)
- `CMD_API_ALL_USER_USAGE` — Usage for all users (includes default domain; `?inode=yes` for inode counts) [id=495, 888, 2185]
- `CMD_API_USER_EXISTS` — Check if a username exists [id=1274]
- `CMD_API_USER_PASSWD` — Change a user's password [id=736]
- `CMD_API_SHOW_USER_CONFIG` — User limits and config (supports `?show=user.conf` and `?show=user.usage`) [id=389, 1422, 1367]
- `CMD_API_SHOW_USER_USAGE` — User resource usage [id=435]
- `CMD_API_SHOW_USER_DOMAINS` — User domains with stats [id=143, 2982]
- `CMD_API_CHANGE_INFO` — Change user email address
- `CMD_API_CHANGE_DOMAIN` — Rename a domain [id=694]
- `CMD_API_EDIT_USER_MESSAGE` — Edit user-level message/notice [id=870]
- `CMD_API_COMMENTS` — Account comments [id=1759]
- `CMD_API_LOGIN_TEST` — Test login credentials [id=530]
- `CMD_API_PUBLIC_STATS` — Public server stats [id=1224]
- `CMD_API_RESELLER_STATS` — Reseller stats [id=368]
- `CMD_API_ADMIN_STATS` — Full admin server statistics

**Packages**
- `CMD_API_PACKAGES_RESELLER` — List/view reseller packages [id=117]
- `CMD_API_PACKAGES_USER` — List/view user packages [id=117]
- `CMD_API_MANAGE_RESELLER_PACKAGES` — Create/modify/delete reseller packages [id=583]
- `CMD_API_MANAGE_USER_PACKAGES` — Create/modify/delete user packages [id=583]

**Domain Management**
- `CMD_API_DOMAIN` — Create, modify, delete domains [id=498]
- `CMD_API_SHOW_DOMAINS` — List user domains [id=336]
- `CMD_API_ADDITIONAL_DOMAINS` — Domain pointers/aliases (includes type, local_mail, php selector, multi-ip info) [id=693, 889, 1063, 1309, 1356, 1684, 1738, 2153, 2427, 2506]
- `CMD_API_DOMAIN_OWNERS` — Map domains to their owners (supports `?domain=single`) [id=579, 1684]
- `CMD_API_DOMAIN_POINTER` — Domain pointer management (full output with local email flag) [id=382, 2506]
- `CMD_API_SUBDOMAIN` / `CMD_API_SUBDOMAINS` — List/create/delete subdomains (aliased; `?domain=all` for all domains; `?list_docroots=yes` for document roots) [id=330, 1589, 1914, 3042]
- `CMD_API_REDIRECT` — Redirect management [id=525]
- `CMD_API_CUSTOM_HTTPD` — Custom HTTPD config [id=580]

**DNS**
- `CMD_API_DNS_ADMIN` — Admin-level DNS zone management (raw save, info, DNSSEC re-sign) [id=504, 531, 532, 626, 1365, 2600]
- `CMD_API_DNS_CONTROL` — User-level DNS control [id=504, 626]
- `CMD_API_DNS_MX` — MX record management (Reseller can call even if user has it disabled) [id=699, 1437]
- `CMD_API_NAME_SERVER` — Nameserver management [id=859]
- `CMD_API_MULTI_SERVER` — Multi-server/clustering management [id=1361]
- DNSSEC API [id=1535] — Full DNSSEC key management

**Email**
- `CMD_API_POP` — Virtual POP accounts (list, create, delete; `?action=full_list`; quota; per-email send limits) [id=229, 643, 1475, 1505, 2827]
- `CMD_API_EMAIL_FORWARDERS` — Email forwarder management [id=411]
- `CMD_API_EMAIL_FILTER` — Email filter/rules management [id=634]
- `CMD_API_EMAIL_AUTH` — Email authentication (SPF/DKIM etc.) [id=588]
- `CMD_API_EMAIL_VACATION` — Vacation/autoresponder (also `CMD_EMAIL_ACCOUNT_VACATION`) [id=348, 684, 1568, 1785, 2713]
- `CMD_API_EMAIL_USAGE` — Email send usage stats (all sends) [id=1675]
- `CMD_EMAIL_ACCOUNT_QUOTA` — Per-account email quota management [id=643, 1568]
- `CMD_CHANGE_EMAIL_PASSWORD` — Change virtual email password (no DA session required) [id=229, 1568]
- `CMD_API_CHANGE_FTP_PASSWORD` — Change FTP password [id=1568]
- `CMD_API_TICKET` — Ticket/messaging system (includes reply count) [id=515, 1037, 1915]
- `CMD_API_TICKET_MANAGE` — Ticket management [id=515, 1915]
- `CMD_API_TICKETS` — List tickets [id=866]
- `CMD_API_RESEND_EMAIL` — Resend notification email [id=782]

**FTP**
- `CMD_API_FTP` — FTP account management (returns custom path; `?extended=yes` for extra info; JSON support) [id=350, 670, 1988, 2174]

**Databases**
- `CMD_API_DATABASES` — Database management (list, create, delete, password change) [id=335, 467, 503, 1104]
- `CMD_API_DB_USER` — Database user management (create, modify; list users for all databases) [id=544, 1880]
- `CMD_API_DB_USER_PRIVS` — Database user privilege management [id=1170]

**Backup & Restore**
- `CMD_API_ADMIN_BACKUP` — Admin-level backup management [id=1090]
- `CMD_API_USER_BACKUP` — User-level backup (with option to suppress notification) [id=1419, 665]
- `CMD_API_SITE_BACKUP` — Site backup/restore (`action=restore` returns result) [id=741]
- Custom hooks: `cmd_site_backup_pre.sh`, `cmd_user_backup_pre.sh` [id=2519]

**SSL/TLS (Legacy)**
- `CMD_API_SSL` — SSL certificate management (includes key in CSR request output) [id=514, 2489]

**File Manager (Legacy)**
- `CMD_API_FILE_MANAGER` — File manager actions (upload without multipart; json_dirs/json_files/json_all actions; type field) [id=652, 740, 1878, 1879, 1958]
- `CMD_FILE_MANAGER` — Human-facing file manager (exists check: `?action=exists&json=yes`; zip/unzip support) [id=2165, 2251]
- `CMD_API_DU_BREAKDOWN` — Disk usage breakdown [id=1730]
- `CMD_DU_BREAKDOWN` / `CMD_BANDWIDTH_BREAKDOWN` — Disk/bandwidth breakdown with cache [id=1258, 648, 747]

**System & Server**
- `CMD_API_ADMIN_FILE_EDITOR` — Admin file editor [id=758]
- `CMD_API_SHOW_SERVICES` — Service status (`?all_info=yes` for memory and PIDs) [id=547, 1980]
- `CMD_API_SYSTEM_INFO` — System information [id=973]
- `CMD_API_PROCESS_MONITOR` — Process monitor [id=1362]
- `CMD_API_LOG_VIEWER` — Log viewer [id=1360]
- `CMD_API_MAIL_QUEUE` — Mail queue management [id=783]
- `CMD_API_IP_MANAGER` — IP address management [id=673]
- `CMD_API_IP_CONFIG` — IP configuration [id=849]
- `CMD_API_SHOW_RESELLER_IPS` — List reseller IPs with status [id=575]
- `CMD_API_LICENSE` — License info [id=1218]
- `CMD_API_HANDLERS` — Handler (CGI/PHP handler) management [id=858]
- `CMD_API_PERL_MODULES` — Perl module management [id=787]
- `CMD_API_PHP_SAFE_MODE` — PHP safe mode settings [id=1230]
- `CMD_API_SPAMASSASSIN` — SpamAssassin configuration [id=731]
- `CMD_API_DIRECTADMIN_CONF` — Read/write directadmin.conf settings [id=1426]
- `CMD_API_EXEC` — Execute arbitrary shell command as the user [id=657] — also has `api_exec_pre.sh` hook [id=3029]

**Crons**
- `CMD_API_CRON_JOBS` — Cron job management [id=364]

**Login Keys (Legacy)**
- `CMD_API_LOGIN_KEYS` — Login key management (included in core_functions feature set) [id=1298, 2898]

**Skin & Appearance**
- `CMD_API_SKINS` — Skin management [id=584]
- `CMD_JSON_OPTIONS` — JSON skin options [id=2399]

**Miscellaneous**
- `CMD_API_SSH_KEYS` — SSH key management [id=2203]
- `CMD_API_SHOW_RESELLER_CONFIG` — Show reseller configuration [id=2450]
- `CMD_API_BANDWIDTH_BREAKDOWN` — Bandwidth breakdown by month [id=377, 648, 747]
- `CMD_API_SHOW_SERVICES` — Show service status [id=547]

### Via Browser DevTools

For any feature visible in the DirectAdmin UI that isn't in the Swagger spec or above list:

1. Log in to the DirectAdmin UI
2. Open DevTools → Network tab
3. Perform the action in the UI
4. Inspect POST requests — note the endpoint URL, request body fields, and response format
5. The Evolution skin's **Live API Documentation** (built-in Swagger UI at `/CMD_EVOLUTION_SKIN?action=api`) is also useful for interactive new-API exploration

### Via Server Debug Mode (Legacy Only)

```bash
# Show only request info (DA compiled after April 1, 2020)
./directadmin debug 188 2>&1 | grep CMD

# Or enable cmd-only debug flag
./directadmin set debug_only_cmd 1
```

Note: Server debug mode only applies to the legacy `CMD_*` codebase, not the new `/api/` endpoints.

---

## Key Decisions When Using the APIs

| Scenario | Use |
|----------|-----|
| Database management | New API (`/api/db-*`) |
| User creation/deletion | Legacy (`CMD_API_ACCOUNT_*`, `CMD_API_SELECT_USERS`) |
| List users/resellers/admins | Legacy (`CMD_API_SHOW_*`) |
| User config/limits | Legacy (`CMD_API_SHOW_USER_CONFIG`) |
| Email accounts | Legacy (`CMD_API_POP`) |
| WordPress | New API (`/api/wordpress/*`) |
| File management | New API (`/api/filemanager*`) |
| TLS/SSL | New API (`/api/domain-tls/*`, `/api/server-tls/*`) |
| Server software builds | New API (`/api/custombuild/*`) |
| Impersonation | Both support it (header-based for new, same for legacy) |
| Responses as JSON (legacy) | Add `?json=yes` to any `CMD_API_` URL |
| Decoding legacy URL-encoded output | Use PHP `parse_str()` or Python `urllib.parse.parse_qs()` |
