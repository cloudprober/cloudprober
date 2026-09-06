NAVY="#0F2A43"; TEAL="#0FA3B1"; AMBER="#E8A33D"
DISH=f'<path d="M48.2 91.47 A52 52 0 0 0 104.53 147.8" fill="none" stroke="{NAVY}" stroke-width="14" stroke-linecap="round"/>'
def mount(w):
    return (f'<g stroke="{NAVY}" stroke-width="{w}" stroke-linecap="round" fill="none">'
            f'<line x1="82" y1="143" x2="82" y2="156"/>'
            f'<line x1="82" y1="156" x2="64" y2="176"/>'
            f'<line x1="82" y1="156" x2="100" y2="176"/>'
            f'<line x1="60" y1="176" x2="104" y2="176"/></g>')
SIG=(f'<circle cx="81.6" cy="114.4" r="5.5" fill="{TEAL}"/>'
     f'<circle cx="98.6" cy="97.4" r="5.5" fill="{TEAL}"/>'
     f'<circle cx="123" cy="73" r="13" fill="none" stroke="{AMBER}" stroke-width="6"/>'
     f'<circle cx="123" cy="73" r="4.5" fill="{TEAL}"/>')
# favicon: heavier dish, one hop, solid target
FAV=(f'<path d="M48.2 91.47 A52 52 0 0 0 104.53 147.8" fill="none" stroke="{NAVY}" stroke-width="19" stroke-linecap="round"/>'
     f'<g stroke="{NAVY}" stroke-width="14" stroke-linecap="round" fill="none">'
     f'<line x1="82" y1="144" x2="82" y2="157"/>'
     f'<line x1="82" y1="157" x2="62" y2="177"/>'
     f'<line x1="82" y1="157" x2="102" y2="177"/>'
     f'<line x1="58" y1="177" x2="106" y2="177"/></g>'
     f'<circle cx="90" cy="106" r="8" fill="{TEAL}"/>'
     f'<circle cx="122" cy="74" r="15" fill="{AMBER}"/>')

def mark(mw=11): return DISH+mount(mw)+SIG

def svg(body, box, bg=None, mono=None, pad=0.0):
    import re
    x0,y0,x1,y1=box; w,h=x1-x0,y1-y0
    p=max(w,h)*pad
    x0-=p; y0-=p; x1+=p; y1+=p; w,h=x1-x0,y1-y0
    if mono: body=re.sub(r'#[0-9A-Fa-f]{6}',mono,body)
    rect=f'<rect x="{x0}" y="{y0}" width="{w}" height="{h}" fill="{bg}"/>' if bg else ''
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{x0:.2f} {y0:.2f} {w:.2f} {h:.2f}">'
            f'{rect}{body}</svg>')
