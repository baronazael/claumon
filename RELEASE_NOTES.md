## Token and cost totals were counted about twice

- **Fixed: tokens and costs were overcounted.** Claude Code logs each part
  of a response (thinking, text, each tool call) as its own line, and every
  line repeats the usage for the whole response. claumon added up every
  line, so token counts and estimated costs were about 2x too high. Each
  response is now counted once. This affected the sessions table, Today,
  History and the hourly heatmap since the first release. Thanks to
  @baronazael for finding and fixing it (#2).

- **Rate-limit gauges and forecasts were never affected.** They come from
  Anthropic's usage API, not from session logs.

- **What you'll see after updating:** token and cost numbers drop by about
  half. At startup claumon rebuilds daily history for every day whose
  session files are still on disk. Older days stay overcounted until they
  age out of the retention window, so the History chart may show a drop
  where your oldest session files begin.

- **History no longer skips days after a restart.** The startup rebuild ran
  alongside other startup work that also writes to claumon's database, and
  some days could fail with "database is locked" and keep stale numbers.
  Writes now wait for each other, so every day gets rebuilt.

- **Homebrew updates now restart claumon.** If you run claumon as a
  background service, the update command in the dashboard now restarts it
  after `brew upgrade`. Before, the service kept running the old version,
  and on macOS it could end up stopped. For this update, restart it
  yourself if claumon doesn't come back:
  `launchctl kickstart -k gui/$(id -u)/com.claumon.dashboard` (macOS) or
  `systemctl --user restart claumon` (Linux).
