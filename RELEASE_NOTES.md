## Forecast and live status show up right after startup

- **The forecast no longer goes missing after a restart.** At startup claumon
  fits the forecast model to your usage history, which takes several seconds
  on a long history. When the first usage check finished before the fit, the
  dashboard showed no forecast until the next check, up to two minutes later.
  The forecast now appears as soon as the fit is done, without an extra
  request to the usage API.

- **"Connected" right away.** After every page load the dashboard showed a
  red dot and "Connecting..." for up to 30 seconds, until the first live
  update arrived. It now connects immediately.

- **Chart axis labels fit.** Daily Tokens axis labels in the hundreds of
  millions were cut off ("910.6M" showed as ")10.6M"). Axis ticks now drop
  the decimal at that size ("911M").

- **New screenshots** in the README and on the website.
