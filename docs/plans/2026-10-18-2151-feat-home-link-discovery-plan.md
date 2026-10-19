---
title: Home Link Discovery - Plan
type: feat
date: 2026-10-18
artifact_contract: ce-unified-plan/v1
product_contract_source: ce-plan-bootstrap
execution: code
---

# Home Link Discovery - Plan

## Goal Capsule

- **Objective:** A person who opens the personal home named in issue 29 can open the sf2-themes site in one click, and the existing project `llms.txt` remains the machine-readable summary.
- **Means:** One absolute link to the live site on each home those URLs now serve (KTD1, A1).
- **Authority:** Product Contract requirements win on behavior. Key Technical Decisions win on mechanism. Implementation units follow both.
- **Execution profile:** code
- **Stop conditions:** Stop if `https://douglasjarquin.github.io/` is no longer the Jekyll user site, if `https://www.douglasjarquin.com/` is no longer the Astro blog, or if the link would require a visual redesign.
- **Finishes and ships:** An implementer completes U1, U2, and U3, each in its own repository. The repository owner ships.

---

## Product Contract

### Summary

Add one visible link to `https://douglasjarquin.github.io/sf2-themes/` on the Jekyll user home and on the www home.
Leave the project `llms.txt` in place, and name its public URL in the README.

### Problem Frame

Issue 29 treats `https://douglasjarquin.github.io/` and canonical `https://douglasjarquin.com/` as one User Pages site that does not link to sf2-themes, and it says no `llms.txt` exists.
On 2026-10-18 those URLs are different sites.
The apex returns 301 to `https://www.douglasjarquin.com/`, which is the Astro blog.
The Jekyll user home does not redirect.
Neither home links to the live site.
The project file is already served at `https://douglasjarquin.github.io/sf2-themes/llms.txt` and already names the live site and the GitHub repository.
Host-root `llms.txt` still 404s on github.io, the apex, and www.

### Requirements

**Home link**

- R1. A person who opens `https://douglasjarquin.github.io/` can open `https://douglasjarquin.github.io/sf2-themes/` in one click from the first screen, without scrolling the post archive and without opening the phone menu.
- R2. A person who opens `https://douglasjarquin.com/` or `https://www.douglasjarquin.com/` can open that same live site in one click from the home.
- R3. Those clicks go to the live site, not to `https://github.com/douglasjarquin/sf2-themes`.

**Machine-readable summary**

- R4. `https://douglasjarquin.github.io/sf2-themes/llms.txt` stays served as `text/plain` and keeps naming the live site, the GitHub repository, the CLI, and the catalog.
- R5. The public README names that `llms.txt` URL next to the live-site URL.

### Acceptance Examples

- AE1. Covers R1. Given the Jekyll home at 600px or narrower, when the page loads, an anchor to the live site is visible without opening the menu and without scrolling past the post list.
- AE2. Covers R2 and R3. Given the www home, when the page loads, the header contains `href="https://douglasjarquin.github.io/sf2-themes/"`. A repository link on About or Now does not satisfy this.
- AE3. Covers R4. Given a GET of the project `llms.txt`, the response is 200 `text/plain` and the body still contains the live site URL and the repository URL.

### Scope Boundaries

- Do not restyle the marketing site, the blog, or the Jekyll theme, and do not add artwork.
- Do not edit sitemap, robots, or canonicals.
- Do not add per-theme URLs.
- Do not add a host-root `llms.txt`.
- Do not change the About or Now anchors that already point at the GitHub repository.
- Do not add a Jekyll file, permalink, or directory named `sf2-themes`.

#### Deferred to Follow-Up Work

- A personal-site `llms.txt` that only points at the project file.
- Extra sentences in the project `llms.txt` for the `sf2` command alias or Slack paste text.

---

## Planning Contract

This plan's file lives in `douglasjarquin/sf2-themes`.
U1 is `douglasjarquin/blog`.
U2 is `douglasjarquin/douglasjarquin.github.io`.
Paths on each unit are relative to that unit's repository.
Neither home repository is present in the sf2-themes checkout.

### Key Technical Decisions

- KTD1. **Absolute live-site URL.** Both homes use `https://douglasjarquin.github.io/sf2-themes/`. A root-relative `/sf2-themes/` is the project site on github.io and a 404 on www.
- KTD2. **Jekyll body, not chrome.** The new anchor sits in `index.html` above `ul.post-list`. `_includes/header.html` only lists titled pages, and `.site-nav .trigger` stays hidden at 600px until hover. `_includes/footer.html` renders after all 62 posts.
- KTD3. **No path named `sf2-themes` in the Jekyll repo.** GitHub serves the project site at that user-pages path. A same-named page in the user site would not be what that URL returns.
- KTD4. **Leave the generator alone.** `llmsTxt()` in `web/src/lib/seo.mjs` and the committed `web/public/llms.txt` already match the live file. Do not edit them, `web/public/robots.txt`, or `web/public/sitemap.xml`.

### Assumptions

- A1. Both homes are in scope. The issue named the Jekyll URL and called the apex its canonical, and those URLs now serve different sites. A link on only one of them leaves the other URL in the issue unchanged.
- A2. The README theme-site bullet is the public place to name the `llms.txt` URL. `docs/features/site.md` names the source file `web/public/llms.txt` and does not name the public URL.

### Sequencing

U1, U2, and U3 do not depend on each other.
Each lands in its own repository and its own pull request.

### Risks & Dependencies

- A search for the string `sf2-themes` on www false-passes. `src/pages/about.astro` and `src/pages/now.astro` already link the GitHub repository, and `/now/` is not in the home nav.
- www does not update until `.github/workflows/deploy-pages.yml` publishes `main`. A local `pnpm build` is not the live check.
- The Jekyll user site publishes from `master` and has no CI. The live check is a GET after that build.
- `isCurrent` in the blog header uses `pathname.startsWith`. An absolute URL does not match a local pathname, so the new item is not marked current. Do not change that function to force a match.

---

## Implementation Units

### U1. Link from the www home

- **Goal:** The www home header and footer open the live site in one click.
- **Requirements:** R2, R3. Implements A1.
- **Dependencies:** none
- **Files:** `src/site.config.ts` in `douglasjarquin/blog`
- **Approach:**
  - Add one `menuLinks` entry whose `path` is the absolute live-site URL (KTD1) and whose `title` is `sf2-themes`.
  - Leave `src/components/layout/Header.astro` and `src/components/layout/Footer.astro` unchanged. Both already render `menuLinks`.
  - Leave `src/pages/about.astro` and `src/pages/now.astro` unchanged.
- **Patterns to follow:** the existing `menuLinks` entries in `src/site.config.ts`
- **Execution note:** This repo has no test script. Prove the link from the built home HTML.
- **Test scenarios:**
  - Happy path: the built home contains `href="https://douglasjarquin.github.io/sf2-themes/"` inside the header nav.
  - Edge: the same href is in the footer, because the footer renders the same list.
  - Edge: that item is not `aria-current="page"` on the home.
  - Integration: the built home does not use a root-relative `/sf2-themes/` for this item.
- **Verification:** `pnpm astro check` and `pnpm build` succeed, and the built home contains the exact href.

### U2. Link from the User Pages home

- **Goal:** The Jekyll home shows the live site before the post list.
- **Requirements:** R1, R3. Implements A1 and AE1.
- **Dependencies:** none
- **Files:** `index.html` in `douglasjarquin/douglasjarquin.github.io`
- **Approach:**
  - Add one absolute anchor above `ul.post-list`, inside `div.home` (KTD1, KTD2).
  - Use visible text `sf2-themes` and leave the anchor in the same tab.
  - Do not add a titled page, and do not put the only copy of the link in the header or the footer (KTD2, KTD3).
- **Patterns to follow:** the RSS anchor already in `index.html`
- **Execution note:** This repo has no test runner. The source order is the proof.
- **Test scenarios:**
  - Happy path: `index.html` contains `href="https://douglasjarquin.github.io/sf2-themes/"` before `ul.post-list`.
  - Edge: the anchor is not inside `_includes/header.html`, so it is outside `.trigger` at 600px.
  - Edge: the repo still has no file, permalink, or directory named `sf2-themes`.
- **Verification:** the anchor precedes the post list in `index.html`, and it is not only in the header include.

### U3. Name the existing llms.txt URL

- **Goal:** A reader of the README sees the public `llms.txt` URL beside the live site.
- **Requirements:** R4, R5. Implements A2.
- **Dependencies:** none
- **Files:** `README.md`
- **Approach:**
  - Extend the theme-site bullet so it also names `https://douglasjarquin.github.io/sf2-themes/llms.txt`.
  - Do not edit the generator or the committed SEO files (KTD4).
- **Patterns to follow:** the existing theme-site bullet in `README.md`
- **Test scenarios:** Test expectation: none. This unit adds one documentation sentence. R4 stays locked by `web/tests/unit/seo.test.mjs` and `web/tests/e2e/static-site.spec.mjs`, which this unit does not change.
- **Verification:** the theme-site bullet contains the public `llms.txt` URL, and no file under `web/` changed.

---

## Verification Contract

- Blog, from `douglasjarquin/blog`: `pnpm astro check` and `pnpm build`. The built home contains `href="https://douglasjarquin.github.io/sf2-themes/"`.
- Jekyll, from `douglasjarquin/douglasjarquin.github.io`: read `index.html`. The anchor precedes `ul.post-list` and is not only in `_includes/header.html`.
- This repo: the README theme-site bullet contains `https://douglasjarquin.github.io/sf2-themes/llms.txt`. Run `mise run web:test` only if a file under `web/` changes.
- After publish: GET `https://douglasjarquin.github.io/` and `https://www.douglasjarquin.com/` and require that exact href in the initial HTML. GET `https://douglasjarquin.com/` only to confirm it still 301s to www. GET the project `llms.txt` and require 200 `text/plain`.

---

## Definition of Done

- U1, U2, and U3 each meet their verification.
- Both published homes contain the exact live-site href in the initial HTML.
- The project `llms.txt` body is unchanged and still returns 200 `text/plain`.
- The diff has no restyle, no new artwork, and no sitemap, robots, or canonical edits.
- Abandoned attempt code is not in the diff.

---

## Sources

- Issue 29: the home link and a documented `llms.txt` that names the live site and the repository. Sitemap, robots, canonicals, and per-theme URLs are out of scope.
- Live checks on 2026-10-18: apex 301 to www, both homes lack the live-site href, project `llms.txt` is 200 `text/plain`, host-root `llms.txt` is 404 on all three origins.
- `douglasjarquin/blog` `src/site.config.ts` `menuLinks`, rendered by `src/components/layout/Header.astro` and `src/components/layout/Footer.astro`. `public/CNAME` is `www.douglasjarquin.com`.
- `douglasjarquin/douglasjarquin.github.io` `index.html`, `_includes/header.html`, and `_includes/footer.html`.
- This repo: `web/src/lib/seo.mjs` `llmsTxt()`, `web/public/llms.txt`, and the theme-site bullet in `README.md`.
