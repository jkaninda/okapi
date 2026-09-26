# Okapi documentation

The documentation site for [Okapi](https://github.com/jkaninda/okapi), built with
[Docusaurus](https://docusaurus.io) and published to <https://okapi.jkaninda.dev>.

## Layout

| Path                             | What it holds                                              |
|----------------------------------|------------------------------------------------------------|
| `docs/`                          | The current version, v1.0.0, served at the site root       |
| `versioned_docs/version-0.11.0/` | The archived v0.11.0 docs, served under `/0.11.0/`         |
| `sidebars.ts`                     | Navigation for the current version                        |
| `versioned_sidebars/`            | Navigation for each archived version                       |
| `docusaurus.config.ts`           | Site configuration, versions, search, redirects            |

## Working on the docs

```bash
npm install     # once
npm start       # dev server with hot reload on http://localhost:3000
npm run build   # production build into build/, fails on broken links
npm run serve   # serve the production build
```

Pages are CommonMark (`markdown.format: 'detect'`), so Go generics and struct
tags in prose are not parsed as JSX. Internal links point at the target `.md`
file so the build can verify them.

## Cutting a new version

Freeze the current docs as a version, then keep editing `docs/` for the next one:

```bash
npm run docusaurus docs:version 1.0.0
```

Deployment runs from `.github/workflows/deploy-docs.yml` (manual dispatch,
choose the branch or tag to publish).
