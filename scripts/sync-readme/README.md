# sync-readme

Refreshes the generated blocks of the profile README from Strapi (`cms.hadihz.me`) and GitHub.
Standard library only; runs daily from `.github/workflows/profile-sync.yml`.

| Marker | Source | Output |
|---|---|---|
| `EXPERIENCE` | `/api/experiences`, featured, not yet ended | the latest role, under "What am I up to?" |
| `EDUCATION` | `/api/educations`, featured | list, newest first |
| `SKILLS` | `/api/tags`, `isSkill=true`, featured | SVG banner in `assets/generated/skills/` |
| `PROJECTS` | `/api/projects`, featured, active first, max 6 | SVG cards in `assets/generated/projects/` |
| `SOCIALS`, `CONTACT` | `/api/profile` social fields; a link whose host does not match its platform is skipped | SVG buttons in `assets/generated/socials/` |
| `STATS` | GitHub GraphQL with `GITHUB_TOKEN` | SVG banner in `assets/generated/stats/` |

Every section fails soft: a non-200, a timeout (10s per request) or an empty result keeps
the previous block and prints a `::warning::`. The exit code stays 0.

## Run locally

From the repository root:

```sh
STRAPI_URL=https://cms.hadihz.me STRAPI_TOKEN=... GITHUB_TOKEN=$(gh auth token) go run ./scripts/sync-readme
go test ./scripts/sync-readme
```

Flags: `-readme README.md`, `-assets assets/generated`, `-site https://hadihz.me`.

Tag icons come from the public Iconify API and are inlined into the SVGs, since an image
cannot load external resources. If Iconify fails, chips fall back to a coloured dot.

## Token

A Strapi custom API token with `find` and `findOne` on Project, Experience, Education
and Tag only. Store it and the URL as the `STRAPI_TOKEN` and `STRAPI_URL` repository secrets.
