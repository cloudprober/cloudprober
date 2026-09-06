import sys, os, re, json
HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(HERE))
OUT  = os.path.join(REPO, "docs", "brand")
sys.path.insert(0, HERE)
from mark import NAVY, TEAL, AMBER, DISH, mount, SIG, FAV, mark
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.pens.boundsPen import BoundsPen
from fontTools.misc.transform import Transform

FONT=os.path.join(HERE, "IBMPlexSans-600.ttf")
WORD="cloudprober"; SPLIT=5; TRACK=-0.010
MARKBOX=(41,57,139,181.5); MW=MARKBOX[2]-MARKBOX[0]; MH=MARKBOX[3]-MARKBOX[1]
GEO_X=(MARKBOX[0]+MARKBOX[2])/2; GEO_Y=(MARKBOX[1]+MARKBOX[3])/2
INK_X, INK_Y = 78.66, 136.46          # measured ink centroid
K=3.2; GAPF=0.95; OPT=0.6; STACK_GAPF=0.55; MOUNT=10

_ft=TTFont(FONT); _gs=_ft.getGlyphSet(); _cm=_ft.getBestCmap(); _upm=_ft["head"].unitsPerEm
CAPR=_ft["OS/2"].sCapHeight/_upm; XHR=_ft["OS/2"].sxHeight/_upm

def word(size):
    """Outline WORD at font size `size`, baseline y=0. -> (run_cloud, run_prober, lsb, xmax, ymax)"""
    sc=size/_upm; x=0.0; runs=[[],[]]; lsb=None; xmax=-1e9; ymax=-1e9
    for i,ch in enumerate(WORD):
        gn=_cm[ord(ch)]
        p=SVGPathPen(_gs); _gs[gn].draw(TransformPen(p, Transform(sc,0,0,-sc,x,0)))
        runs[0 if i<SPLIT else 1].append(p.getCommands())
        b=BoundsPen(_gs); _gs[gn].draw(b)
        if b.bounds:
            a,bb,c,dd=b.bounds
            if lsb is None: lsb=x+a*sc
            xmax=max(xmax,x+c*sc); ymax=max(ymax,dd*sc)
        x += _gs[gn].width*sc + TRACK*size
    return " ".join(runs[0]), " ".join(runs[1]), lsb, xmax, ymax

def frame(body, box, bg=None, pad=0.04):
    x0,y0,x1,y1=box; p=max(x1-x0,y1-y0)*pad
    x0-=p; y0-=p; x1+=p; y1+=p
    rect=f'<rect x="{x0:.2f}" y="{y0:.2f}" width="{x1-x0:.2f}" height="{y1-y0:.2f}" fill="{bg}"/>' if bg else ''
    return ('<svg xmlns="http://www.w3.org/2000/svg" '
            f'viewBox="{x0:.2f} {y0:.2f} {x1-x0:.2f} {y1-y0:.2f}">{rect}{body}</svg>')

def recolor(s, ink=None, accent=None, target=None):
    if ink:    s=s.replace(NAVY, ink)
    if accent: s=s.replace(TEAL, accent)
    if target: s=s.replace(AMBER, target)
    return s

def horizontal(size=100, twotone=True, ink=NAVY, accent=TEAL, mark_ink=NAVY,
               mark_accent=TEAL, mark_target=AMBER, bg=None, pad=0.04):
    cap=CAPR*size; s=(K*cap)/MH; gap=GAPF*cap
    c,p,lsb,xmax,ymax = word(size)
    wx = MW*s + gap - lsb
    baseline = ((GEO_Y+OPT*(INK_Y-GEO_Y))-MARKBOX[1])*s + cap/2
    mk = recolor(mark(MOUNT), mark_ink if mark_ink!=NAVY else None,
                 mark_accent if mark_accent!=TEAL else None,
                 mark_target if mark_target!=AMBER else None)
    tx = (f'<path d="{c}" fill="{ink}"/><path d="{p}" fill="{accent if twotone else ink}"/>')
    body=(f'<g transform="translate({-MARKBOX[0]*s:.3f} {-MARKBOX[1]*s:.3f}) scale({s:.5f})">{mk}</g>'
          f'<g transform="translate({wx:.3f} {baseline:.3f})">{tx}</g>')
    return frame(body,(0,0,wx+xmax,max(MH*s, baseline+ymax)), bg, pad)

def stacked(size=100, twotone=True, ink=NAVY, accent=TEAL, mark_ink=NAVY,
            mark_accent=TEAL, mark_target=AMBER, bg=None, pad=0.04):
    cap=CAPR*size; s=(K*cap)/MH
    c,p,lsb,xmax,ymax = word(size)
    ww=xmax-lsb; mkw=MW*s; total=max(mkw,ww)
    mkx=(total-mkw)/2 - (INK_X-GEO_X)*s
    wx=(total-ww)/2 - lsb
    baseline = MH*s + STACK_GAPF*cap + cap
    mk = recolor(mark(MOUNT), mark_ink if mark_ink!=NAVY else None,
                 mark_accent if mark_accent!=TEAL else None,
                 mark_target if mark_target!=AMBER else None)
    tx=(f'<path d="{c}" fill="{ink}"/><path d="{p}" fill="{accent if twotone else ink}"/>')
    body=(f'<g transform="translate({mkx-MARKBOX[0]*s:.3f} {-MARKBOX[1]*s:.3f}) scale({s:.5f})">{mk}</g>'
          f'<g transform="translate({wx:.3f} {baseline:.3f})">{tx}</g>')
    return frame(body,(min(0,mkx),0,total,baseline+ymax), bg, pad)

def square(b):
    x0,y0,x1,y1=b; s=max(x1-x0,y1-y0); cx,cy=(x0+x1)/2,(y0+y1)/2
    return (cx-s/2,cy-s/2,cx+s/2,cy+s/2)
