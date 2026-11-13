# AGENTS.md

FINANIKA — a Goravel (Laravel-style Go) app: hand-rolled JWT + Casbin RBAC, live
market-data JSON APIs, a Gemini/Groq chat advisor, and one server-rendered dashboard.
**No git repository is initialised.** Never propose commits, branches, or PRs.

Generic Goravel/Laravel conventions: `.agents/skills/goravel-development/SKILL.md`.
This file only covers repo-specific traps.

## Commands

```powershell
go run .                                        # http://127.0.0.1:3000 (APP_PORT)
go run . artisan migrate                        # apply migrations
go run . artisan make:migration create_x_table  # auto-registers in bootstrap/migrations.go
go build ./... ; go vet ./...                   # the verification pair
go test ./...                                   # whole suite
go test ./tests/feature -run TestExampleTestSuite
go test ./app/services                          # pure unit tests, no DB
```

`artisan` / `artisan.ps1` / `artisan.bat` wrap `go run . artisan "$@"`; no standalone binary.
`.air.toml` builds into `tmp/main.exe`.

**Do not run `gofmt -w .`.** Scaffold files are intentionally unformatted and will always
appear in `gofmt -l`: `app/facades/*.go`, `app/http/controllers/user_controller.go`,
`config/{ai,database,grpc,http,telemetry}.go`, `routes/grpc.go`, `bootstrap/providers.go`.
Format only files you edit.

## Traps that waste time

- **Module path is doubled on purpose**: `edukasi-finansial-goedukasi-finansial-go`.
- **`binding:"..."` tags are silently ignored** (`gin.DisableBindValidation()` at boot).
  Use `ctx.Request().Validate(rules)` through the shared `validate()` helper in
  `auth_controller.go` (returns `(errors, ok)`, renders 422). Rules: `required`, `email`,
  `integer`, `numeric`, `gte`, `lte`, `min_len`.
- **`Query().First()` returns `nil` when no row matches** — the framework swallows
  `gorm.ErrRecordNotFound`. Use `FirstOrFail()` or check the destination's zero value.
- **`os.Getenv` never sees `.env`** — Goravel loads it into viper without `os.Setenv`.
  Use `config.Env(...)` in `config/*.go` or `facades.Config().GetString(...)`. Real OS env
  vars work and take precedence.
- **Views are parsed once at boot** — any view edit needs a server restart.
- **Every file under `resources/views` is a Go template**; a stray `{{` in CSS/JS breaks
  boot. The callable name is the `{{ define "x" }}` name if present, else the filename.
- **Global 3 s request-timeout middleware** from `http.request_timeout`; slow handlers need
  `HTTP_REQUEST_TIMEOUT` raised (currently `90`, HTTP client `60s`).
- **`go run .` leaves an orphan child on port 3000.** Kill by port:
  ```powershell
  Get-NetTCPConnection -LocalPort 3000 -State Listen | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }
  ```

## Wiring

`main.go` → `bootstrap.Boot()` → `bootstrap/app.go` (migrations, routes, providers, config).
`routes/web.go` serves HTML/static; `routes/api.go` serves JSON under `/api`.

```
app/http/controllers/   auth, profile, market, chat, risk, portfolio, dashboard, admin, education
app/http/middleware/    auth_middleware.go, role_middleware.go
app/models/             User, ChatMessage, RiskProfile, Portfolio, education/admin models
app/services/           market, ai, insight, world_asset, chat_context, casbin, market_cache
app/facades/            generated wrappers — import these, not the framework's
config/                 one file per key; casbin_model.conf + casbin_policy.csv
bootstrap/migrations.go explicit migration registry
```

Middleware = struct with `Signature() string` + `Handle(http.Context)`, attached via
`route.Middleware(...)`. `AuthMiddleware` decodes the JWT and stashes `user_id` /
`user_role`; read the id with `currentUserID(ctx)`.

`GET /platform` only 302s to `/dasbor#/platform`. Auth links use `/dasbor#/masuk` or `/dasbor#/daftar` to open the modal. The dashboard is one view: a new page is a `data-panel`
plus a hash address, never a second route with its own sidebar/topbar/script.

## Auth & RBAC

JWT is hand-rolled (`golang-jwt/v5` + `bcrypt`) in `auth_controller.go`, not the framework
guard. Claims `sub`, `role`, `iat`, `exp`, HS256; middleware pins the algorithm.
`JWT_SECRET` / `JWT_TTL` in `config/jwt.go`. Login also sets an HttpOnly
`FINANIKA_token` cookie (API clients still use Bearer); `POST /api/logout` clears it.
Web paths `/pembelajaran`, `/belajar`, `/kuis`, `/profil-risiko`, `/portofolio`, and
`/asisten-ai` require auth; `/admin` and `/admin/{path}` require auth + admin. Guests
redirect to `/dasbor#/masuk`, non-admins to `/dasbor?error=admin_only`. The dashboard
shows the admin denial notice from the query string and also guards hash pages.

Casbin is used directly (`github.com/casbin/casbin/v2`; `github.com/goravel/casbin` does not
exist). Enforcer is a `sync.Once` singleton in `app/services/casbin.go` built from
`config/casbin_model.conf` + `config/casbin_policy.csv`. `RoleMiddleware` allows a route's
explicit role **or** a Casbin `(role, path, method)` match. Policy edits need a restart.

The login modal has a UI-only "Lupa Password?" form; it validates the email and shows a
confirmation but does not call an API or send mail yet.

New users default to `user`; self-registration cannot create admins. Seeded accounts:
`admin@finanika.com` and `user@finanika.com`, both via `upsertSeedUser` (re-hashes bcrypt,
updates existing row). `go run . artisan db:seed --seeder=AdminUserSeeder` repairs a login.

## Profile

`PUT /api/profile` (auth) accepts JSON or multipart, never an email from the body, and
normalises gender to `Laki-laki`/`Perempuan` (blank allowed). Bio ≤100 runes. Avatar sniffed
with `http.DetectContentType` (JPG/PNG/WEBP, ≤2 MB), stored under `public/uploads/avatars/`,
served by the existing `/public` route. `userResource` and `adminUserRow` must both expose
`gender`, `avatar_url`, `bio`.

## Admin content

`data-panel="admin"`; all routes behind `AuthMiddleware` + `RoleMiddleware("admin")` (users
get 403): `/api/admin/dashboard`, CRUD for `/modules`, `/quizzes`, `/platforms`.

- Modules store slides as JSON via `models.SlidesToJSON`; delete cascades to quizzes and
  `user_module_progress`. Quiz answers `a`–`d`; all four options + explanation required.
- Platform pros/cons are JSON arrays, ≥3 per side, each side containing a cost word
  (`fee`/`biaya`/`spread`/`komisi`/`admin`); `cleanAdminPlatform` enforces it.
- One reused `#admin-modal`; table rows use delegated listeners.
- Admin strings go in both `TEXT_I18N_EXTRA` and `TEXT_I18N_DYNAMIC`.
- `EducationController` serves three random questions per attempt and shuffles options;
  option ids stay `a`–`d`, the letter is display-only. Pass score `total/2 + 1`. Every
  submit also inserts a `quiz_results` row (score/total/passed); module completion still
  lives in `user_module_progress`. Seeders are idempotent (fill-only/backfill) so admin edits
  survive `db:seed`.
- `GET /api/platforms` is public (guests can see Rekomendasi Platform).
- NLG evaluation is admin-only at `GET/POST /api/nlg-test`; rows live in
  `nlg_test_results` and the panel lists the latest 100. `GET /pembelajaran` redirects to
  `/dasbor#/belajar`; dashboard "Pelajari" buttons open the matching education module.

## Market data

Three upstreams fetched independently: CoinGecko (BTC), xaus.com (gold), Yahoo Finance
(USD/IDR). Per-source failures go under `prices.errors`; only all-fail fails the request.

- **Yahoo 429s without a browser-like `User-Agent`** — `fetchJson` sets one; keep it.
- One-hour cache via `facades.Cache().Remember` (memory), keys prefixed with `APP_NAME`.
- **Memory cache is backed by `market_cache`**: every successful fetch is also stored as
  JSON. On 429/503 the stale row is returned instead of 502 (`GetCurrentPrices`,
  `cachedQuotes`, `fullDailySeries`, `getHourlyHistory`). Fallback does not refresh
  `cached_at` or the memory cache, so the next request retries. No row → normal failure.
- Gold has no daily comparison from xaus.com; `fetchGold` borrows 24 h % from Yahoo `GC=F`.
- Shared helpers `toMap`/`toSlice`/`toFloat64`/`toStr` live in `app/services`.
  `fetchJson` decodes objects; CoinGecko `/coins/markets` is an array → `fetchJsonArray`.
- Listings share `cachedQuotes` (hourly cache + `market_cache` fallback); a new listing
  should use it. Return 502 with a friendly `error` only if upstream fails **and** no cache
  copy exists.
- **Every Yahoo quote uses one shape**: `yahooQuoteURL`
  (`/v8/finance/chart/%s?interval=1d&range=1d`), read by `fetchYahooQuote` from chart `meta`:
  live `regularMarketPrice`, previous close `chartPreviousClose`. Indices keep `^`.
- `/api/crypto` (protected): ten coins from `cryptoCoins`, market-cap desc; each coin's
  Yahoo ticker is `cryptoCoin.Yahoo` (Sui `SUI20947-USD`). Sui replaced Dogecoin.
- `/api/forex` (protected): ten currencies vs IDR in `forexCurrencies` order (also the Kurs
  cycle order). Ten Yahoo requests are concurrent but gated by `yahooQuoteWorkers` (6); a
  missing pair is skipped, all-fail is 502. `fetchYahooIdrRate` is shared with USD/IDR.
- `/api/saham` (protected): twenty rows (IDX + US), `market` = `id`/`us`.
- `/api/komoditas` (protected): sixteen Yahoo futures + four Bank Indonesia PIHPS food rows
  (chili/shallot/garlic/rawit). PIHPS rows have no Yahoo ticker; `PIHPSID` marks them and
  `fetchPihpsQuotes` uses four workers. `fetchPihpsJson` sends a browser UA,
  `Referer: bi.go.id/hargapangan`, and `X-Requested-With`. Prices are IDR/kg; format with
  `moneyIdr`/`commodityValue`, never USD conversion. `change_percent` uses the latest two
  points. First PIHPS row is the national average.
- `name`/`unit` come from our tables, not Yahoo. Yahoo quotes coffee/grains in US cents
  (`meta.currency == "USX"`), so `yahooPriceScale` divides by 100 (also for history).
  `MTF=F` stopped updating (null closes since 26 Dec 2025); `chartDomain` un-pins the axis.
- `/api/market/history?asset=<coin|gold|usd>&days=1..3650` (protected): daily closes.
  CoinGecko `market_chart` up to 365 days; longer and all gold/USD use Yahoo `GC=F` /
  `USDIDR=X` / coin ticker. `days` clamped; bad asset 422; upstream failure 502. PIHPS
  assets use `fetchPihpsSeries` in 30-day chunks (`pihpsChunkWorkers` 4). `source` is what
  actually served the series (`coingecko`/`yahoo_finance`); do not reintroduce `HistorySource`.
- **History caches the sliced result under `market_history_{asset}_{days}`, but upstream is
  only `market_history_full_{asset}_{1y|10y}` (1 h).** The 3H…10T sweep costs one Yahoo +
  one CoinGecko request, not eight. CoinGecko anonymous tier rejects `days > 365` with 401;
  the Yahoo fallback absorbs the 429s from a cold click-through of all ten coins.
- History uses `fetchJsonWithRetry` (3 attempts, 1.2 s/2.4 s) on 429/503; live uses plain
  `fetchJson`. Yahoo bars are dated with `meta.gmtoffset`. `chartDomain()` pins
  `xaxis.min/max` to `now - days*86400000 … now` so all assets share a date scale.
- `/api/world-assets` (auth): `WorldAssetService` owns the twenty-row table
  (`worldAssetSeeds`), with live quantities on gold/silver/Bitcoin/Ethereum. Prices come from
  existing paths; a failed price keeps the estimate and `live: false`, so this endpoint
  never 502s. Sort after applying live prices. Cached one hour under `market_world_assets`.

## AI (Chat FINANIKA)

Gemini REST at `{base_url}/models/{model}:generateContent`, `x-goog-api-key`, body
`systemInstruction` + `contents` + `generationConfig`; reply is
`candidates[0].content.parts[].text`.

- **`gemini-1.5-flash` and `gemini-2.5-flash` are rejected for new keys.** Default
  `gemini-3.8-flash`, overridable via `GEMINI_MODEL`, set in four places (`config/gemini.go`,
  `defaultGeminiModel`, `.env`, `.env.example`) — keep in sync. Empty key or
  `geminiApiKeyPlaceholder` means "not configured".
- Answers are plain text. `plainTextInstruction` forbids markdown and is appended to all
  three system prompts; `dashboard.html` also runs `cleanMarkdown()`. Assign AI text with
  `textContent`, never `innerHTML`.
- Every answer closes with a `Saran:` line via `penutupInstruction`, which **forbids**
  "ini hanya edukasi" / "bukan nasihat keuangan profesional" / "bukan saran investasi".
  Do not add a disclaimer back into a prompt.
- Persona and per-risk-profile DCA rules live in `systemInstruction()`, including the
  off-topic rule. Every prompt carries the latest `aiNewsDigestLimit` (5) headlines via
  `formatNewsDigest()` (titles only, cached list).
- **`GenerateWithFallback` tries Gemini then Groq on every Gemini failure** (429/503,
  timeout, 5xx, unknown model, unset key), except `ErrAIPromptRejected` (a content decision
  must not be re-asked elsewhere). Each switch is logged. All AI features enter here:
  `GenerateChat`, `GenerateNarasi`, `MarketSummary`, `AnalyzeAsset`, `ExplainNews`.
- Groq: `config/groq.go`, `GROQ_API_KEY`, `GROQ_MODEL`, `GROQ_BASE_URL`. OpenAI-style POST
  to `{base_url}/chat/completions`; read `choices[0].message.content` (ignore `reasoning`).
  `max_tokens` = `groqMaxTokens` (4096), `reasoning_effort: low`; system prompt is its own
  `system` message. **Model is `openai/gpt-oss-120b`**; `llama-3.3-70b-versatile` is
  retired (404).
- Chain bounded by `aiFallbackBudget` (80 s), Gemini leg `geminiTimeout` (12 s). After a
  failure `markGeminiCooldown()` skips Gemini for `geminiCooldown` (3 min). Output caps are
  4096; do not slice the returned string.
- Both providers failing returns friendly `429`/`503` sentinels
  (`ErrAIRateLimited`/`ErrAIHighDemand`) via `aiFailure()`, never upstream JSON. Other
  failures keep `502 {"error","detail"}`; sentinel bodies stay free of `detail`.
- `/api/ai/summary` (auth): `{summary, profile, generated_at, cached, prices}`.
  `MarketSummary` caches 30 min per risk profile **and language**
  (`ai_market_summary_<profil>_<lang>`); failures are never cached, `cached` is read before
  `Remember`, `prices` is fetched outside the cache. The Dasbor card is the only AI card and
  deliberately does not go through `/api/chat/send`.
- `/api/ai/analyze-asset` (auth): `{asset,label,days,history}` (≥5, ≤400 points); cached 1 h
  per asset (`ai_analysis_<asset>`). No page calls it.
- `/api/insight` (auth): three Google News RSS queries (Saham/SBSN/Logam Mulia), 1 h
  cache, falls back to `contohInsightNews` (32 items) when the merged feed has fewer than
  `insightMinItems` (30), a category is missing, or `INSIGHT_FEED=false`. Page shows
  `insightMaxItems` (32). `/api/insight/explain` resolves by `id` then `title` against the
  currently served list; explanations cache 30 min per article **and language**
  (`ai_news_explanation_<lang>_<id>`). `splitNewsTitle` strips the RSS `<source>` suffix
  before the last-dash fallback.
- **Chat panel**: `SendMessage` reads `recentChatHistory(userID)` before storing the new
  message and replays the newest `ChatHistoryTurns` (6) turns. `cleanChatHistory` drops
  invalid roles and the `[gambar]` placeholder, caps each turn at 600 runes. History comes
  from the DB, not the browser.
- `/api/chat/send` accepts `{message, context}`; `PrepareChatContext` cleans labels, drops
  bad prices, caps counts. **Prices, profile and news come from the server**
  (`mergeChatPrices`, `chatNewsFromInsight`, `latestRiskProfileRecord`); only platforms are
  trusted from the client (they come from `state.platforms`). `chatPlatformContext` attaches
  a description only for the platform named in the question (first sentence, ≤160 runes).
  `GenerateChat` inherits the full persona — do not fork it.
- **Images**: multipart `message`, `context` (JSON string), `image`; `bindChatRequest`
  branches on `Content-Type`. `services.NewChatImage` sniffs MIME (PNG/JPEG/WEBP, ≤4 MB
  `ChatImageMaxBytes`); the view downscales to ≤1280 px / ~1.2 MB. Only Gemini sees images —
  `GenerateChatImage` has **no Groq leg** and uses `geminiVisionTimeout` (60 s). Image-only
  messages persist the literal `[gambar]`.
- `SendMessage` persists the user message before calling the provider; on failure no
  assistant row is written.

## Risk profile & portfolio

`risk_profiles` stores one row per submission; `latestRiskProfile(userID)` reads newest by
`id desc`, default `Moderat`. Thresholds in `risk_controller.go`:
`skorMaksimalKonservatif = 10`, `skorMaksimalModerat = 15`; accepted score 5–20
(5 questions × 1–4). Changing the quiz means changing these constants.

`GET /api/risk/profile` returns `{has_profile, profile|null}`; `has_profile` distinguishes
"never took the quiz" from a Moderat result. `POST /api/risk/quiz` rejects scores outside 5–20.

`portfolios` = one row per holding, all routes auth + scoped by `user_id`.
`GET /api/portfolio` always returns an array. `POST` accepts
`{asset_symbol, quantity, buy_price}`; only symbols in `models.AssetTypeForSymbol` are
accepted (unknown → 422), and `asset_type` is derived server-side. `quantity`/`buy_price`
use `required|numeric` plus an explicit `> 0` check (fractions valid). `DELETE` always
filters by `id` **and** `user_id`; `RowsAffected == 0` → 404. Prices are computed only on
the frontend from `/api/market`, `/api/crypto`, `/api/saham`, `/api/komoditas`.

## Frontend

Two views: `resources/views/landing.html` at `/` for guests, and `resources/views/dashboard.html` at `/dasbor` by `DashboardController`. Tailwind
via `cdn.tailwindcss.com` and fonts via `fonts.bunny.net`; without internet it renders
unstyled, not blank.

- Auth token in `localStorage.FINANIKA_token` (legacy `fingo_token` is also accepted by the landing redirect; `fingo_user`, `fingo_profile`), sent as
  `Authorization: Bearer`; 401 clears the session and market cards show "Harga terkunci".
  `fingo_portfolio` is gone — the portfolio is server-backed; do not reintroduce it.
- **Language + theme.** `localStorage.finanika_lang` drives `applyLanguage()`; two layers:
  `data-i18n` + `I18N` for chrome, and generated `public/i18n-content.js`
  (`window.TEXT_I18N`) for panel content, augmented by `TEXT_I18N_EXTRA`,
  `TEXT_I18N_DYNAMIC`, `PATTERN_I18N`, `RANGE_I18N`. `localizeDom()` keeps the Indonesian
  source in `__i18nSrc`; a `MutationObserver` re-runs it. Text split by an inline span needs
  `data-i18n-block`. **AI text is never translated** (`data-no-i18n`). The same
  `state.language` goes to summary `?language=`, chat context, and insight-explain body.
  Theme lives in `localStorage.finanika_theme` / `html[data-theme]`; light mode is a CSS
  override block, not second markup. `refreshChartsForTheme()` updates `chartRegistry`.
- **All chart times are WIB (UTC+7), not browser-local.** `chartDisplayStamp` adds
  `WIB_OFFSET_MS`; every `xaxis` sets `datetimeUTC: true`; `chartWibLabel` uses `getUTC*`;
  `chartTooltipFormat` appends " WIB"; `chartDomain()` applies the same shift. Do not use a
  browser-local formatter.
- Panels are lists/tables fed by the four listing APIs (no dummy data), loaded on first
  visit and after login via `loadAccountData()`. Crypto/Saham/Komoditas share the
  list|chart + education layout; grid is `items-stretch`, both cards `h-full` (keep bottom
  edges aligned — `deltaBottom` 0). Saham/Komoditas have client-side tabs (`state.sahamTab`,
  `state.komoditasTab`) via `paintSahamTabs` / `paintKomoditasTabs` and
  `ensure*TabSelection()`. `ROW_CHARTS[id].tbody` accepts one id or an array.
- `ASSET_API` maps only dashboard symbols (BTC/XAU/USD); coin ids pass through. `chartStyle`
  supplies color/format; `PAGE_ASSETS` needs an entry per new asset or rupiah prices get the
  coin formatter.
- **Crypto** chart follows the coin list (`state.cryptoAsset`, `selectChartRow`); the old
  BTC hero card is gone — do not reintroduce a static label. "Tentang <koin>" uses
  `COIN_ABOUT` (static, no Gemini).
- **Kurs** is driven by `state.forexAsset` (card, chart, education, converter); ←/→ wrap the
  ten rows. Converter swaps by reordering DOM; `state.convSource` preserves the typed side.
  The old static USD/IDR card ids are gone.
- **Saham** splits rows by `market` into `saham-id-tbody` / `saham-us-tbody`; indices use
  `indexPoints`, IDR rows `idr0`, others `usdValue`.
- **Komoditas** rows are USD via `usdValue`; PIHPS rows use `moneyIdr`/`commodityValue`. The
  education card links to `finance.yahoo.com/quote/`.
- **CSV buttons read state, not a new request** (`state.crypto.data`, `state.forex.data`)
  and export raw numbers via `csvNumber` (dot decimal), not formatted strings.
- **Semua Pasar** (`data-panel="pasar"`) aggregates the four loaders in `Promise.all`;
  `PASAR_GROUPS` sets order, `state.pasarCategory` maps asset → group. Detail opens
  `#pasar-modal` (a `manual: true` `ROW_CHARTS` entry, no `EDU_PANELS`); only the
  `[data-detail]` button opens it — row clicks do nothing.
- **Aset Dunia** uses `/api/world-assets`; `state.world` feeds table + donut. `worldValue()`
  is the single formatter (USD triliun / IDR kuadriliun); the donut rebuilds on currency
  toggle only while visible.
- **Portfolio/DCA** share `ASSET_CATALOG`; `unitPriceIdr()` reads loaded state. A new asset
  must be added there **and** to `portfolioAssetTypes`.
- **Insight Global** cards from `/api/insight`; "Tanya FinGo AI" posts `{id,title,language}`;
  `state.insightAnswers[id|language]` avoids repeat requests.
- **Chat panel** has two modes (`openChat('full'|'floating')`), one DOM. Full mode uses
  `--chat-left`/`--chat-top` so it does not cover sidebar/topbar; z-index 35 (under sidebar
  50 and mobile drawer 40). `showPage()` minimizes a full chat; `askFinGo()` is the single
  path into chat. Mic uses `SpeechRecognition` (`id-ID`) with `.recording`; clip uses
  `state.chatImage` and posts `FormData` (no manual `Content-Type`).
- **Platforms** (`data-panel="platform"`) are DB-backed via public `GET /api/platforms` into
  `state.platforms`; two pill groups filter in place. Logos use Google favicon services with
  a `data-fallback-src` retry; keep the letter tile underneath.
- `renderCrypto`/`renderForex`/`renderSaham`/`renderKomoditas`/`renderPortfolio` distinguish
  empty states (not logged in vs failed vs empty) — keep that. Commodity table prefix is
  `komo`, not `komoditas`. `changeCell()` is shared.

## Testing & data

`tests/feature/example_test.go` is the only feature test and is trivial. `tests.TestCase`
calls `bootstrap.Boot()` in `init()`, so feature tests need a reachable DB and a valid
`.env` — not hermetic. Use testify `suite` + `tests.TestCase` for new ones.
`app/services/ai_context_test.go` holds pure unit tests for AI context caps.

MySQL (`DB_CONNECTION=mysql`, `goravel/mysql`), database `db_edukasi_go`, alongside XAMPP.
Local CLI: `C:\xampp\mysql\bin\mysql.exe -u root`. Migrations are listed explicitly in
`bootstrap/migrations.go`.
