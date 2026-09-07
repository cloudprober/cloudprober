#!/usr/bin/env python3
"""
Build the complete Cloudprober logo package from geometry + one font file.

    pip install fonttools cairosvg pillow numpy
    python3 tools/logogen/build_logo.py --verify

Rebuilds docs/brand in place by default; --out writes elsewhere. The published
copies the site actually serves live in docs/static and are updated separately.

Everything downstream — 17 SVGs, PNG rasters, favicon.ico, apple-touch icon,
maskable PWA icon, social card, web manifest — is derived from the constants in
GEOMETRY and TYPE below. Change one and rebuild; nothing goes out of sync.

If the instanced font is missing, pass --instance-from with the Google Fonts
variable release (IBMPlexSans[wdth,wght].ttf) and it will be cut at weight 600.
"""

import argparse, json, math, os, re, sys

# ---------------------------------------------------------------- palette ---

NAVY  = "#0F2A43"   # ink: dish, mount, "cloud"
TEAL  = "#0FA3B1"   # signal: probe dots, target centre, "prober"
AMBER = "#E8A33D"   # target ring
WHITE = "#FFFFFF"

# --------------------------------------------------------------- geometry ---
# The dish is a circular arc, centre of curvature (100, 96), radius 52, rim at
# 185deg and 85deg -> a 100deg opening symmetric about a 135deg optical axis
# (45deg above horizontal on screen). The dish vertex is (63.23, 132.77).
#
# Every element of the ray sits exactly on that axis. Probe dot 1 is at r/2 =
# 26 from the vertex, which is the focal point of a spherical reflector of this
# radius -- it is the feed horn, not a decorative dot. If you move or add
# anything on the ray, keep it on the 45deg line from the vertex.

DISH_PATH  = "M48.2 91.47 A52 52 0 0 0 104.53 147.8"
DISH_C     = (100.0, 96.0)
DISH_R     = 52.0
AXIS_DEG   = 135.0                      # SVG coords, y down
VERTEX     = (DISH_C[0] + DISH_R * math.cos(math.radians(AXIS_DEG)),
              DISH_C[1] + DISH_R * math.sin(math.radians(AXIS_DEG)))

def on_axis(dist):
    """Point at `dist` from the dish vertex along the optical axis."""
    a = math.radians(AXIS_DEG - 180.0)
    return (VERTEX[0] + dist * math.cos(a), VERTEX[1] + dist * math.sin(a))

FOCUS = on_axis(DISH_R / 2)             # (81.61, 114.39) -- probe dot 1

# Full mark: dish stroke 14, mount stroke 10, two probe dots, ring target.
MARK_DISH_SW  = 14
MARK_MOUNT_SW = 10
# Icon: everything thicker, one probe dot, solid target disc. The post top is
# 144 (not 141): at stroke 14 the round cap reaches 7 units past the endpoint,
# and the dish inner surface at x=82 sits at y=135.38. A top of 141 puts the
# cap at 134 and breaks through into the bowl. Re-check this if either stroke
# weight changes -- round caps move the visible tip when you re-weight.
ICON_DISH_SW  = 19
ICON_MOUNT_SW = 14
ICON_POST_TOP = 144

MARKBOX = (41, 57, 139, 181.5)          # tight bbox of the full mark, incl. strokes
ICONBOX = (38, 54, 140, 184)
INK_CENTROID = (78.66, 136.46)          # measured, not eyeballed; mass is low-left

# ------------------------------------------------------------------- type ---

WORD  = "cloudprober"
SPLIT = 5                               # "cloud" | "prober" for the two-tone
TRACK = -0.010                          # em

# ---------------------------------------------------------- lockup metrics ---

K           = 3.2       # mark height as a multiple of cap height
GAPF        = 0.95      # mark-to-wordmark gap, multiple of cap height
OPT         = 0.60      # optical correction toward the ink centroid, 0..1
STACK_GAPF  = 0.55      # vertical gap in the stacked lockup
PAD         = 0.04      # viewBox padding, fraction of the long edge

# ============================================================= mark drawing ==

def _dish(sw):
    return (f'<path d="{DISH_PATH}" fill="none" stroke="{NAVY}" '
            f'stroke-width="{sw}" stroke-linecap="round"/>')

def _mount(sw, post_top=143, post_bot=156, splay=18, foot=176, base=(60, 104)):
    return (f'<g stroke="{NAVY}" stroke-width="{sw}" stroke-linecap="round" fill="none">'
            f'<line x1="82" y1="{post_top}" x2="82" y2="{post_bot}"/>'
            f'<line x1="82" y1="{post_bot}" x2="{82-splay}" y2="{foot}"/>'
            f'<line x1="82" y1="{post_bot}" x2="{82+splay}" y2="{foot}"/>'
            f'<line x1="{base[0]}" y1="{foot}" x2="{base[1]}" y2="{foot}"/></g>')

def _signal():
    d1 = FOCUS
    d2 = on_axis(50.02)
    tg = on_axis(84.52)
    return (f'<circle cx="{d1[0]:.1f}" cy="{d1[1]:.1f}" r="5.5" fill="{TEAL}"/>'
            f'<circle cx="{d2[0]:.1f}" cy="{d2[1]:.1f}" r="5.5" fill="{TEAL}"/>'
            f'<circle cx="{tg[0]:.0f}" cy="{tg[1]:.0f}" r="13" fill="none" '
            f'stroke="{AMBER}" stroke-width="6"/>'
            f'<circle cx="{tg[0]:.0f}" cy="{tg[1]:.0f}" r="4.5" fill="{TEAL}"/>')

def mark():
    """Full mark. Use at 48px and above."""
    return _dish(MARK_DISH_SW) + _mount(MARK_MOUNT_SW) + _signal()

def icon():
    """Simplified reduction. Use below 48px -- see the note on ICON_POST_TOP."""
    d = on_axis(37.85)
    t = on_axis(83.11)
    return (_dish(ICON_DISH_SW)
            + _mount(ICON_MOUNT_SW, post_top=ICON_POST_TOP, post_bot=157,
                     splay=20, foot=177, base=(58, 106))
            + f'<circle cx="{d[0]:.0f}" cy="{d[1]:.0f}" r="8" fill="{TEAL}"/>'
            + f'<circle cx="{t[0]:.0f}" cy="{t[1]:.0f}" r="15" fill="{AMBER}"/>')

def recolor(s, ink=None, accent=None, target=None):
    if ink:    s = s.replace(NAVY, ink)
    if accent: s = s.replace(TEAL, accent)
    if target: s = s.replace(AMBER, target)
    return s

def frame(body, box, bg=None, pad=PAD):
    x0, y0, x1, y1 = box
    p = max(x1 - x0, y1 - y0) * pad
    x0, y0, x1, y1 = x0 - p, y0 - p, x1 + p, y1 + p
    rect = (f'<rect x="{x0:.2f}" y="{y0:.2f}" width="{x1-x0:.2f}" '
            f'height="{y1-y0:.2f}" fill="{bg}"/>') if bg else ""
    return ('<svg xmlns="http://www.w3.org/2000/svg" '
            f'viewBox="{x0:.2f} {y0:.2f} {x1-x0:.2f} {y1-y0:.2f}">{rect}{body}</svg>')

def square(b):
    x0, y0, x1, y1 = b
    s = max(x1 - x0, y1 - y0)
    cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
    return (cx - s / 2, cy - s / 2, cx + s / 2, cy + s / 2)

# =============================================================== wordmark ====

class Wordmark:
    """Outlines WORD to paths. No live text ever reaches the output."""

    def __init__(self, font_path):
        from fontTools.ttLib import TTFont
        self.ft   = TTFont(font_path)
        self.gs   = self.ft.getGlyphSet()
        self.cm   = self.ft.getBestCmap()
        self.upm  = self.ft["head"].unitsPerEm
        self.capr = self.ft["OS/2"].sCapHeight / self.upm
        self.xhr  = self.ft["OS/2"].sxHeight / self.upm

    def runs(self, size):
        """-> (cloud_path, prober_path, left_bearing, x_max, y_max), baseline y=0."""
        from fontTools.pens.svgPathPen import SVGPathPen
        from fontTools.pens.transformPen import TransformPen
        from fontTools.pens.boundsPen import BoundsPen
        from fontTools.misc.transform import Transform
        sc = size / self.upm
        x, out, lsb, xmax, ymax = 0.0, [[], []], None, -1e9, -1e9
        for i, ch in enumerate(WORD):
            gn = self.cm[ord(ch)]
            p = SVGPathPen(self.gs)
            self.gs[gn].draw(TransformPen(p, Transform(sc, 0, 0, -sc, x, 0)))
            out[0 if i < SPLIT else 1].append(p.getCommands())
            b = BoundsPen(self.gs); self.gs[gn].draw(b)
            if b.bounds:
                a, _, c, dd = b.bounds
                if lsb is None: lsb = x + a * sc
                xmax = max(xmax, x + c * sc)
                ymax = max(ymax, dd * sc)
            x += self.gs[gn].width * sc + TRACK * size
        return " ".join(out[0]), " ".join(out[1]), lsb, xmax, ymax

# ================================================================ lockups ====

def _mk(mark_ink, mark_accent, mark_target, s, dx=0.0):
    m = recolor(mark(),
                mark_ink if mark_ink != NAVY else None,
                mark_accent if mark_accent != TEAL else None,
                mark_target if mark_target != AMBER else None)
    return (f'<g transform="translate({dx - MARKBOX[0]*s:.3f} '
            f'{-MARKBOX[1]*s:.3f}) scale({s:.5f})">{m}</g>')

def horizontal(wm, size=100, twotone=True, ink=NAVY, accent=TEAL,
               mark_ink=NAVY, mark_accent=TEAL, mark_target=AMBER, bg=None):
    mh  = MARKBOX[3] - MARKBOX[1]
    mw  = MARKBOX[2] - MARKBOX[0]
    cap = wm.capr * size
    s   = (K * cap) / mh
    c, p, lsb, xmax, ymax = wm.runs(size)
    wx  = mw * s + GAPF * cap - lsb
    geo_y = (MARKBOX[1] + MARKBOX[3]) / 2
    baseline = ((geo_y + OPT * (INK_CENTROID[1] - geo_y)) - MARKBOX[1]) * s + cap / 2
    body = (_mk(mark_ink, mark_accent, mark_target, s)
            + f'<g transform="translate({wx:.3f} {baseline:.3f})">'
              f'<path d="{c}" fill="{ink}"/>'
              f'<path d="{p}" fill="{accent if twotone else ink}"/></g>')
    return frame(body, (0, 0, wx + xmax, max(mh * s, baseline + ymax)), bg)

def stacked(wm, size=100, twotone=True, ink=NAVY, accent=TEAL,
            mark_ink=NAVY, mark_accent=TEAL, mark_target=AMBER, bg=None):
    mh  = MARKBOX[3] - MARKBOX[1]
    mw  = MARKBOX[2] - MARKBOX[0]
    cap = wm.capr * size
    s   = (K * cap) / mh
    c, p, lsb, xmax, ymax = wm.runs(size)
    ww, mkw = xmax - lsb, mw * s
    total   = max(mkw, ww)
    geo_x   = (MARKBOX[0] + MARKBOX[2]) / 2
    # centre on the ink centroid, not the bbox: the mark's mass is 11.3u left
    mkx = (total - mkw) / 2 - (INK_CENTROID[0] - geo_x) * s
    wx  = (total - ww) / 2 - lsb
    baseline = mh * s + STACK_GAPF * cap + cap
    body = (_mk(mark_ink, mark_accent, mark_target, s, dx=mkx)
            + f'<g transform="translate({wx:.3f} {baseline:.3f})">'
              f'<path d="{c}" fill="{ink}"/>'
              f'<path d="{p}" fill="{accent if twotone else ink}"/></g>')
    return frame(body, (min(0, mkx), 0, total, baseline + ymax), bg)

# ================================================================= output ====

def all_svgs(wm):
    W = WHITE
    return {
        "cloudprober-horizontal.svg":          horizontal(wm),
        "cloudprober-horizontal-onecolor.svg": horizontal(wm, twotone=False),
        "cloudprober-horizontal-mono.svg":     horizontal(wm, twotone=False,
                                                mark_accent=NAVY, mark_target=NAVY),
        "cloudprober-horizontal-white.svg":    horizontal(wm, twotone=False, ink=W,
                                                mark_ink=W, mark_accent=W, mark_target=W),
        "cloudprober-horizontal-ondark.svg":   horizontal(wm, ink=W, mark_ink=W),
        "cloudprober-stacked.svg":             stacked(wm),
        "cloudprober-stacked-onecolor.svg":    stacked(wm, twotone=False),
        "cloudprober-stacked-mono.svg":        stacked(wm, twotone=False,
                                                mark_accent=NAVY, mark_target=NAVY),
        "cloudprober-stacked-white.svg":       stacked(wm, twotone=False, ink=W,
                                                mark_ink=W, mark_accent=W, mark_target=W),
        "cloudprober-stacked-ondark.svg":      stacked(wm, ink=W, mark_ink=W),
        "cloudprober-mark.svg":                frame(mark(), square(MARKBOX)),
        "cloudprober-mark-mono.svg":           frame(recolor(mark(), None, NAVY, NAVY),
                                                square(MARKBOX)),
        "cloudprober-mark-white.svg":          frame(recolor(mark(), WHITE, WHITE, WHITE),
                                                square(MARKBOX)),
        "cloudprober-icon.svg":                frame(icon(), square(ICONBOX), pad=0.03),
        "cloudprober-icon-mono.svg":           frame(recolor(icon(), None, NAVY, NAVY),
                                                square(ICONBOX), pad=0.03),
        "cloudprober-icon-white.svg":          frame(recolor(icon(), WHITE, WHITE, WHITE),
                                                square(ICONBOX), pad=0.03),
        "cloudprober-icon-ondark.svg":         frame(recolor(icon(), WHITE),
                                                square(ICONBOX), bg=NAVY, pad=0.03),
    }

MANIFEST = {
    "name": "Cloudprober", "short_name": "Cloudprober",
    "icons": [
        {"src": "/favicon-192.png", "sizes": "192x192", "type": "image/png"},
        {"src": "/favicon-512.png", "sizes": "512x512", "type": "image/png"},
        {"src": "/icon-512-maskable.png", "sizes": "512x512",
         "type": "image/png", "purpose": "maskable"},
    ],
    "theme_color": NAVY, "background_color": WHITE, "display": "standalone",
}

def build(out, font):
    import cairosvg
    from PIL import Image
    svgdir, pngdir, favdir = (os.path.join(out, d) for d in ("svg", "png", "favicon"))
    for d in (svgdir, pngdir, favdir):
        os.makedirs(d, exist_ok=True)

    wm = Wordmark(font)
    svgs = all_svgs(wm)
    for n, s in svgs.items():
        assert "<text" not in s and "font-family" not in s, f"live text in {n}"
        with open(os.path.join(svgdir, n), "w") as f:
            f.write(s)

    def rp(src, dst, w, h=None, bg=None):
        cairosvg.svg2png(url=os.path.join(svgdir, src), write_to=dst,
                         output_width=w, output_height=h, background_color=bg)

    for base in ("cloudprober-horizontal", "cloudprober-horizontal-white",
                 "cloudprober-stacked", "cloudprober-mark"):
        for w in (400, 800, 1600):
            rp(f"{base}.svg", os.path.join(pngdir, f"{base}-{w}w.png"), w)

    for sz in (16, 32, 48, 64, 128, 192, 256, 512):
        rp("cloudprober-icon.svg", os.path.join(favdir, f"favicon-{sz}.png"), sz, sz)

    # Every ICO plane must be a true render at its own size. Passing PIL a small
    # source silently drops the larger planes -- that shipped broken once.
    ico_sizes = [16, 32, 48, 64]
    planes = [Image.open(os.path.join(favdir, f"favicon-{s}.png")).convert("RGBA")
              for s in ico_sizes]
    planes[-1].save(os.path.join(favdir, "favicon.ico"), format="ICO",
                    sizes=[(s, s) for s in ico_sizes], append_images=planes[:-1])

    navy = (15, 42, 67, 255)
    # apple-touch must be opaque: iOS composites transparency onto black.
    rp("cloudprober-icon-white.svg", "/tmp/_at.png", 132, 132)
    at = Image.new("RGBA", (180, 180), navy)
    at.alpha_composite(Image.open("/tmp/_at.png").convert("RGBA"), (24, 24))
    at.convert("RGB").save(os.path.join(favdir, "apple-touch-icon.png"))

    # maskable: artwork inside the inner 80% safe zone
    rp("cloudprober-icon-white.svg", "/tmp/_mk.png", 307, 307)
    mk = Image.new("RGBA", (512, 512), navy)
    mk.alpha_composite(Image.open("/tmp/_mk.png").convert("RGBA"), (102, 102))
    mk.convert("RGB").save(os.path.join(favdir, "icon-512-maskable.png"))

    rp("cloudprober-horizontal-ondark.svg", "/tmp/_og.png", 760)
    og = Image.new("RGBA", (1200, 630), navy)
    o = Image.open("/tmp/_og.png").convert("RGBA")
    og.alpha_composite(o, ((1200 - o.width) // 2, (630 - o.height) // 2))
    og.convert("RGB").save(os.path.join(pngdir, "social-card-1200x630.png"))

    with open(os.path.join(out, "site.webmanifest"), "w") as f:
        json.dump(MANIFEST, f, indent=2)

    print(f"built {len(svgs)} svgs + rasters into {out}/")
    return out

# ================================================================= verify ====

def verify(root):
    """Run against the built tree -- or better, against an extracted archive."""
    import cairosvg, numpy as np
    from PIL import Image
    fail = []

    if ICON_POST_TOP != 144:
        fail.append(f"ICON_POST_TOP is {ICON_POST_TOP}; 144 is the value that clears "
                    "the dish at stroke 14 -- re-measure before changing it")
    stale = 'y1="%d"' % 141
    for dirpath, _, names in os.walk(root):
        for n in names:
            if not n.endswith(".svg"):
                continue
            s = open(os.path.join(dirpath, n), errors="ignore").read()
            if stale in s:
                fail.append(f"pre-fix icon post survives in {n}")
            if "<text" in s or "font-family" in s:
                fail.append(f"live text in {n}")

    svgdir = os.path.join(root, "svg")

    def ras(body):
        cairosvg.svg2png(
            bytestring=('<svg xmlns="http://www.w3.org/2000/svg" '
                        f'viewBox="30 45 120 150">{body}</svg>').encode(),
            write_to="/tmp/_v.png", output_width=1600, background_color="white")
        return np.array(Image.open("/tmp/_v.png").convert("L")) < 160

    src = open(os.path.join(svgdir, "cloudprober-icon.svg")).read()
    m = re.search(r'(<g stroke="%s" stroke-width="%d".*?</g>)'
                  % (NAVY, ICON_MOUNT_SW), src, re.S)
    d, mt = ras(_dish(ICON_DISH_SW)), ras(m.group(1))
    prot = sum(1 for c in range(d.shape[1])
               if len(np.where(d[:, c])[0]) and len(np.where(mt[:, c])[0])
               and (np.where(mt[:, c])[0] < np.where(d[:, c])[0].min()).any())
    if prot:
        fail.append(f"icon post breaks the dish surface in {prot} columns")

    ico = Image.open(os.path.join(root, "favicon", "favicon.ico"))
    if sorted(ico.ico.sizes()) != [(16, 16), (32, 32), (48, 48), (64, 64)]:
        fail.append(f"favicon.ico planes: {sorted(ico.ico.sizes())}")
    else:
        for s in (16, 32, 48, 64):
            cairosvg.svg2png(url=os.path.join(svgdir, "cloudprober-icon.svg"),
                             write_to="/tmp/_t.png", output_width=s,
                             output_height=s, background_color=None)
            a = np.array(Image.open("/tmp/_t.png").convert("RGBA")).astype(int)
            b = np.array(ico.ico.getimage((s, s)).convert("RGBA")).astype(int)
            if np.abs(a - b).mean() > 0.01:
                fail.append(f"ICO {s}px plane is not a true render")

    at = Image.open(os.path.join(root, "favicon", "apple-touch-icon.png"))
    if at.size != (180, 180) or at.mode == "RGBA":
        fail.append(f"apple-touch-icon is {at.size} {at.mode}, must be 180x180 opaque")

    print("VERIFY:", "all checks passed" if not fail else "\n  - ".join([""] + fail))
    return not fail

# =================================================================== main ====

def instance_font(var_ttf, dst, weight=600):
    from fontTools.ttLib import TTFont
    from fontTools.varLib.instancer import instantiateVariableFont
    f = TTFont(var_ttf)
    axes = {a.axisTag for a in f["fvar"].axes}
    loc = {"wght": weight}
    if "wdth" in axes:
        loc["wdth"] = 100
    instantiateVariableFont(f, loc).save(dst)
    print(f"instanced {var_ttf} at wght={weight} -> {dst}")
    return dst

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(HERE))

if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    # Defaults are anchored to this file, not the caller's cwd, so that
    # `python3 tools/logogen/build_logo.py` rebuilds docs/brand in place from
    # anywhere in the tree.
    ap.add_argument("--out", default=os.path.join(REPO, "docs", "brand"))
    ap.add_argument("--font", default=os.path.join(HERE, "IBMPlexSans-600.ttf"))
    ap.add_argument("--instance-from", metavar="VARIABLE_TTF",
                    help="cut --font from a Google Fonts variable release first")
    ap.add_argument("--verify", action="store_true")
    a = ap.parse_args()

    if a.instance_from:
        instance_font(a.instance_from, a.font)
    if not os.path.exists(a.font):
        sys.exit(f"font not found: {a.font}\n"
                 "Get IBMPlexSans[wdth,wght].ttf from the Google Fonts repo and pass "
                 "--instance-from, or supply an already-instanced SemiBold.")

    build(a.out, a.font)
    if a.verify:
        sys.exit(0 if verify(a.out) else 1)
