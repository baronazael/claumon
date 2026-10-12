## New claumon logo; clearer update notice

- **The dashboard has a logo.** The claumon mascot now sits next to the name
  in the top bar, and the browser tab icon is the mascot instead of a plain
  colored circle.

- **The tab icon still shows session usage.** A dot in the corner of the
  mascot turns green, yellow or red as session usage crosses 50% and 80%,
  just like the old circle did.

- **The update badge tells you how to update.** It used to be a link to the
  GitHub release page, which did nothing inside the VS Code panel and looked
  like a one-click update. It now reads `vX.Y.Z available`, and hovering it
  shows the command for your install: `claumon update`, or
  `brew update && brew upgrade claumon` for Homebrew installs.

- **`claumon update` no longer overwrites Homebrew installs.** It used to
  replace the binary inside Homebrew's Cellar, leaving Homebrew convinced the
  old version was still installed. It now detects a Homebrew install and
  tells you to run `brew update && brew upgrade claumon` instead.

- **README and website.** The README header and the claumon website use the
  new logo too.
