# Release notes

One file per version, `<version>.md`, named after the git tag (for example `0.6.0-alpha.md`). [`CHANGELOG.md`](../../CHANGELOG.md) keeps the short list of changes; the release notes add what an operator must check or set before upgrading, then describe each change in full.

## Before tagging

1. Write or finish `<version>.md`. Lead with "Before you upgrade": behaviour changes, settings to turn new features on (with the `COLA_REGISTRY_*` and `CDT_REGISTRY_*` env names), and how to roll back.
2. Replace local commit references with PR links.
3. In `CHANGELOG.md`, replace `Unreleased` in the version's heading with the tag date (`YYYY-MM-DD`) and point its link at `<previous>...<version>`.
4. Commit both, and merge to `main` before tagging.

## At tag time

Pushing the tag runs the release job in `.github/workflows/go.yml`. It creates the GitHub release with `--generate-notes`, so its body is GitHub's "What's Changed" list. Once the job has finished, replace the body with the notes file:

```sh
gh release edit <version> -R criteo/command-launcher-registry \
  --notes-file docs/release-notes/<version>.md
```

The HTML comment at the top of a notes file does not render on GitHub. To keep GitHub's generated list as well, copy it from the release before you run the edit and paste it under the notes.
