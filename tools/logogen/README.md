# Logo generator

`build_logo.py` builds the entire Cloudprober logo kit — 17 SVG variants, the PNG
renders, the favicon set, `favicon.ico`, the apple-touch and maskable PWA icons, the
social card and a web manifest — from the geometry constants at the top of the file plus
one font.

```sh
pip install fonttools cairosvg pillow numpy
python3 tools/logogen/build_logo.py --verify
```

Output goes to `docs/brand/` by default; `--out` writes elsewhere. See
`docs/brand/README.md` for the usage rules, colours, minimum sizes, and how to publish a
change through to `docs/static/`.

`IBMPlexSans-600.ttf` is a weight-600 instance of the Google Fonts variable release
(SIL OFL 1.1), used only at build time — the wordmark is outlined to paths, so no output
file carries a font dependency. If you need to cut it again, pass `--instance-from` the
variable `IBMPlexSans[wdth,wght].ttf`.

Do not hand-edit files under `docs/brand/`; change a constant here and rebuild.
