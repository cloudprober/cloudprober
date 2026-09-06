# Logo generator

Geometry and layout code for the Cloudprober logo. The artwork it produces lives in
`docs/brand/`, and the copies the website and favicon set actually serve live in
`docs/static/`. See `docs/brand/README.md` for usage rules, colours and minimum sizes.

- `mark.py` — mark and icon geometry, the source of truth for the drawing.
- `build_final.py` — wordmark outlining and lockup layout helpers.
- `IBMPlexSans-600.ttf` — build input, SIL OFL 1.1.

Note that the driver that writes the kit is not here yet; `build_final.py` currently
provides only the helpers it would call. Keep `mark.py` in step with any hand-edit to a
shipped SVG.
