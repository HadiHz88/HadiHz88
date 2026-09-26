# sync-readme

Refreshes the CMS-driven blocks of the profile README from Strapi (`cms.hadihz.me`).
Standard library only; runs daily from `.github/workflows/profile-sync.yml`.

| Marker | Source | Output |
|---|---|---|
| `EXPERIENCE` | `/api/experiences`, featured, not yet ended | "What am I up to?" list |
| `EDUCATION` | `/api/educations`, featured | list, newest first |
| `SKILLS` | `/api/tags`, `isSkill=true` | shields.io badges by category |
| `PROJECTS` | `/api/projects`, featured, active first, max 6 | SVG cards in `assets/generated/projects/` |

Every section fails soft: a non-200, a timeout (10s per request) or an empty result keeps
the previous block and prints a `::warning::`. The exit code stays 0.

## Run locally

From the repository root:

```sh
STRAPI_URL=https://cms.hadihz.me STRAPI_TOKEN=... go run ./scripts/sync-readme
go test ./scripts/sync-readme
```

Flags: `-readme README.md`, `-assets assets/generated/projects`, `-site https://hadihz.me`.

## Token

A Strapi custom API token with `find` and `findOne` on Project, Experience, Education
and Tag only. Store it and the URL as the `STRAPI_TOKEN` and `STRAPI_URL` repository secrets.
