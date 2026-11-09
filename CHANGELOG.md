# Changelog

What changed in each release of Troye. Installs update to the newest release; see
[docs/deploy.md](docs/deploy.md#updates-and-versions).

## Unreleased

- Nutrition screen, after a few weeks of data: your real daily burn measured from what you
  ate and how your weight moved, with a one-tap budget built on it (and back to the
  formula); nutrients drifting up or down over 4 weeks; and the weekday that runs over.
- Progress tab: the Weight screen now sits next to a new Nutrition screen with a chart per
  nutrient (daily bars, 7-day average, target), warnings when last week's average of
  calories, protein, carbs, fat, fiber, sugar or salt was off target, and estimates for days
  you didn't log or only half logged. The Monday summary names up to two warnings.
- Every published image carries a software bill of materials (SBOM), and an OpenSSF
  Scorecard run checks the repository's security practices every week.
- golang.org/x/net 0.62.0, which fixes GO-2026-6860.
- SECURITY.md: how to report a problem privately, and how quickly fixes ship.
- Security scans on every pull request and every Monday: CodeQL, govulncheck, Grype
  (web and worker dependencies, released images), dependency review and gitleaks.
- The containers' filesystems are read-only apart from `pb_data` and `/tmp`.
- The compose file follows the major version (`:0`) instead of `:latest`, and the
  installer fetches the compose files of the newest release instead of `main`.

## 0.1.0 (2026-11-04)

- Releases: a `vX.Y.Z` tag publishes `:X.Y.Z`, `:X.Y`, `:X` and `:latest`, and installs follow
  `:latest`, so they only update to a release. `main` publishes `:edge`.
- Every published image has a signed provenance attestation (`gh attestation verify`).
- Security and maintenance section in the README.
