# Cloudprober logo

Mark: a dish on a mount, emitting a probe that lands on a target. Wordmark: IBM Plex Sans
SemiBold, outlined to paths — there is no live text and no font dependency in any file.

![preview](PREVIEW.png)

## Colour

| role   | hex       | use                                        |
| ------ | --------- | ------------------------------------------ |
| ink    | `#0F2A43` | dish, mount, `cloud`                       |
| signal | `#0FA3B1` | probe dots, target centre, `prober`        |
| target | `#E8A33D` | target ring                                |

On dark backgrounds the ink becomes `#FFFFFF`; signal and target stay as-is.

## Which file

| situation                                   | file                                       |
| ------------------------------------------- | ------------------------------------------ |
| default, light background                   | `svg/cloudprober-horizontal.svg`           |
| narrow or square space                      | `svg/cloudprober-stacked.svg`              |
| dark background                             | `svg/cloudprober-horizontal-ondark.svg`    |
| single-colour print, engraving, stamps      | `svg/cloudprober-horizontal-mono.svg`      |
| over a photo or colour, knocked out         | `svg/cloudprober-*-white.svg`              |
| avatar, sticker, mark alone                 | `svg/cloudprober-mark.svg`                 |
| favicon, app icon, anything under 48px      | `svg/cloudprober-icon.svg`                 |

`-onecolor` keeps the full-colour mark but sets the whole wordmark in ink — use it when the
two-tone split competes with surrounding colour.

## The icon is not the mark

`cloudprober-icon.svg` is a separate drawing, not a scaled-down `cloudprober-mark.svg`. The
full mark carries two probe dots and a ring target with an open centre. At 16px one device
pixel spans about 8.6 units of the artboard while that ring's counter is 5.5 units wide, so
it fills in and the target turns into a blob. The icon drops to one probe dot, thickens the
dish and mount, and replaces the ring with a solid disc.

**Use the icon below 48px. Use the mark at 48px and above.** Do not scale one to cover the
other's range.

## Clear space and minimum sizes

Clear space on all four sides is one cap height of the wordmark — 0.31× the height of the
horizontal lockup, or 0.22× the height of the stacked lockup.

| asset               | minimum        |
| ------------------- | -------------- |
| horizontal lockup   | 120px wide     |
| stacked lockup      | 90px wide      |
| mark alone          | 48px           |
| icon                | 16px           |

Aspect ratios are fixed: horizontal 2.83:1, stacked 1.32:1, mark and icon 1:1. Never
re-space the mark against the wordmark; the gap is 0.95× cap height and the mark is aligned
to the cap band with a 60% optical correction for its bottom-left weight.

## Favicon wiring

```html
<link rel="icon" href="/favicon.ico" sizes="32x32">
<link rel="icon" href="/cloudprober-icon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="/apple-touch-icon.png">
<link rel="manifest" href="/site.webmanifest">
```

`favicon.ico` carries 16/32/48/64. `apple-touch-icon.png` is 180×180, opaque navy with iOS
padding — it must not be transparent. `icon-512-maskable.png` respects the 80% safe zone for
`purpose: "maskable"` in the web manifest.

Social card: `png/social-card-1200x630.png` for `og:image` and `twitter:image`.

## Where these files come from

This directory holds the artwork. The geometry that generates it lives in
`tools/logogen/`:

| file | role |
| ---- | ---- |
| `tools/logogen/mark.py` | the mark and icon geometry — the actual source of truth |
| `tools/logogen/build_final.py` | lockup layout: outlines the wordmark, spaces it against the mark |
| `tools/logogen/IBMPlexSans-600.ttf` | build input, weight-600 instance of the Google Fonts variable release |

`build_final.py` exposes `horizontal()`, `stacked()`, `frame()`, `recolor()` and `square()`,
and reads `K` (mark-to-cap ratio), `GAPF` (gap), `TRACK` (tracking) and `OPT` (optical
correction). Changing one of those and rebuilding keeps every variant in sync. It needs
`fonttools`; the raster steps need `cairosvg` and `pillow`.

**The driver that calls those helpers and writes the files is not in the repo yet** — the kit
was generated elsewhere and only the helper module came across, so `build_final.py` cannot
currently rebuild this directory on its own. Until that lands, treat the committed artwork as
authoritative and edit `mark.py` alongside any hand-change to a shipped SVG, so the geometry
and the artwork do not drift.

Anything derived from the mark also has published copies under `docs/static/`, which are what
the website and the favicon set actually serve. A change here is not live until those are
updated too.

## Licence

IBM Plex Sans is SIL OFL 1.1. The wordmark is outlined, so nothing needs to ship the font,
and the OFL reserved-font-name clause does not apply to outlined artwork.
