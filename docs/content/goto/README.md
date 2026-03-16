# URL Shortener (`/goto/<keyword>`)

This directory contains short URL redirects for `cloudprober.org/goto/<keyword>`.

Each `.md` file (except `_index.md` and this README) defines a redirect from
`cloudprober.org/goto/<keyword>` to a destination URL.

## Adding a new shortlink

Create a file `<keyword>.md` in this directory:

```markdown
---
title: "<keyword>"
dest: "<destination URL>"
sitemap_exclude: true
---
```

For example, to add `cloudprober.org/goto/docs` → `https://cloudprober.org/docs/`:

```markdown
---
title: "docs"
dest: "https://cloudprober.org/docs/"
sitemap_exclude: true
---
```

## How it works

Hugo builds each content file using the layout in `docs/layouts/goto/`. The
layout outputs a minimal HTML page with a `<meta http-equiv="refresh">` tag that
performs a client-side redirect. This works on any static hosting including
GitHub Pages.

## Verifying

After adding a shortlink, build the site and check the output:

```bash
cd docs && npm install && node_modules/.bin/hugo/hugo
cat public/goto/<keyword>/index.html
```

The generated HTML should contain a meta refresh tag pointing to your destination URL.
