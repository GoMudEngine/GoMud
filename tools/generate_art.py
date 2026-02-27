#!/usr/bin/env python3
"""Generate DOGMud branding art assets using Pillow.

Produces:
  - favicon.png  (32x32 pixel-art crescent moon)
  - web_bg.png   (800x600 dark fantasy moons scene)
"""

import math
import os
import random

from PIL import Image, ImageDraw

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)
IMAGES_DIR = os.path.join(
    PROJECT_ROOT, "_datafiles", "html", "public", "static", "images"
)


# ---------------------------------------------------------------------------
# Favicon: 32x32 crescent moon on dark background
# ---------------------------------------------------------------------------
def generate_favicon():
    size = 32
    img = Image.new("RGBA", (size, size), (10, 10, 25, 255))
    draw = ImageDraw.Draw(img)

    # Draw a silver/white circle (the moon)
    cx, cy, r = 16, 15, 11
    draw.ellipse(
        [cx - r, cy - r, cx + r, cy + r],
        fill=(210, 215, 230, 255),
    )

    # Cut a dark circle out of the upper-right to make a crescent
    cut_r = 10
    cut_cx, cut_cy = cx + 6, cy - 4
    draw.ellipse(
        [cut_cx - cut_r, cut_cy - cut_r, cut_cx + cut_r, cut_cy + cut_r],
        fill=(10, 10, 25, 255),
    )

    # Add a few tiny stars
    stars = [(5, 5), (27, 4), (3, 24), (26, 22), (14, 3)]
    for sx, sy in stars:
        img.putpixel((sx, sy), (200, 200, 240, 255))

    # Small bright dot for The Eye
    draw.ellipse([25, 8, 28, 11], fill=(240, 240, 255, 255))

    out_path = os.path.join(IMAGES_DIR, "favicon.png")
    img.save(out_path)
    print(f"  Wrote {out_path}")


# ---------------------------------------------------------------------------
# Web background: 800x600 dark night sky with three moons
# ---------------------------------------------------------------------------
def generate_web_bg():
    W, H = 800, 600
    img = Image.new("RGB", (W, H))
    draw = ImageDraw.Draw(img)

    # Sky gradient: deep navy at top to dark blue-black near ground
    for y in range(H):
        t = y / H
        r = int(5 + 15 * t)
        g = int(5 + 10 * t)
        b = int(30 - 15 * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))

    # Stars
    random.seed(42)
    for _ in range(250):
        sx = random.randint(0, W - 1)
        sy = random.randint(0, int(H * 0.7))
        brightness = random.randint(140, 255)
        tint = random.choice([
            (brightness, brightness, brightness + 15),
            (brightness, brightness - 10, brightness),
            (brightness + 10, brightness + 10, brightness - 5),
        ])
        img.putpixel((sx, sy), tint)
        # Some brighter stars get a 2x2 block
        if brightness > 220 and sx < W - 1 and sy < H * 0.7 - 1:
            img.putpixel((sx + 1, sy), tint)
            img.putpixel((sx, sy + 1), tint)

    # --- Swiftmoon (large, left-ish) ---
    _draw_moon(draw, img, cx=200, cy=120, radius=55,
               color=(200, 210, 230), glow_color=(100, 110, 150))

    # --- The Wanderer (medium, center-upper) ---
    _draw_moon(draw, img, cx=450, cy=85, radius=30,
               color=(180, 170, 200), glow_color=(90, 80, 120))

    # --- The Eye (small but bright, right-ish) ---
    _draw_moon(draw, img, cx=620, cy=150, radius=15,
               color=(240, 240, 255), glow_color=(140, 140, 180),
               bright=True)

    # Silhouetted landscape (mountain/treeline)
    ground_color = (5, 5, 10)

    # Mountains
    peaks = [
        (0, 520), (80, 440), (180, 480), (280, 420), (380, 460),
        (450, 410), (530, 450), (620, 430), (720, 470), (800, 500),
    ]
    # Draw filled polygon for mountains
    mountain_points = peaks + [(W, H), (0, H)]
    draw.polygon(mountain_points, fill=(8, 8, 15))

    # Treeline (jagged silhouette on top of mountains)
    tree_y_base = 480
    tree_points = [(0, H)]
    for x in range(0, W + 5, 5):
        # Get mountain height at this x
        base_y = _interp_peaks(peaks, x)
        # Add tree-like spikes
        spike = random.randint(5, 30)
        ty = base_y - spike
        tree_points.append((x, ty))
    tree_points.append((W, H))
    draw.polygon(tree_points, fill=ground_color)

    # Ground fill
    draw.rectangle([0, int(H * 0.92), W, H], fill=ground_color)

    out_path = os.path.join(IMAGES_DIR, "web_bg.png")
    img.save(out_path)
    print(f"  Wrote {out_path}")


def _draw_moon(draw, img, cx, cy, radius, color, glow_color, bright=False):
    """Draw a moon with a soft glow."""
    W, H = img.size

    # Outer glow (multiple expanding translucent circles)
    for i in range(6, 0, -1):
        gr = radius + i * 8
        alpha_factor = (7 - i) / 7.0 * 0.15
        glow_r = int(glow_color[0] * alpha_factor)
        glow_g = int(glow_color[1] * alpha_factor)
        glow_b = int(glow_color[2] * alpha_factor)

        for y in range(max(0, cy - gr), min(H, cy + gr)):
            for x in range(max(0, cx - gr), min(W, cx + gr)):
                dist = math.sqrt((x - cx) ** 2 + (y - cy) ** 2)
                if dist <= gr:
                    fade = 1.0 - (dist / gr)
                    pr, pg, pb = img.getpixel((x, y))
                    nr = min(255, pr + int(glow_r * fade))
                    ng = min(255, pg + int(glow_g * fade))
                    nb = min(255, pb + int(glow_b * fade))
                    img.putpixel((x, y), (nr, ng, nb))

    # Moon disc
    draw.ellipse(
        [cx - radius, cy - radius, cx + radius, cy + radius],
        fill=color,
    )

    # Subtle craters / surface detail
    random.seed(cx * 100 + cy)
    for _ in range(max(3, radius // 5)):
        crx = cx + random.randint(-radius + 4, radius - 4)
        cry = cy + random.randint(-radius + 4, radius - 4)
        crr = random.randint(2, max(3, radius // 8))
        if math.sqrt((crx - cx) ** 2 + (cry - cy) ** 2) + crr < radius - 2:
            shade = tuple(max(0, c - random.randint(15, 35)) for c in color)
            draw.ellipse(
                [crx - crr, cry - crr, crx + crr, cry + crr],
                fill=shade,
            )

    # The Eye gets a bright center highlight
    if bright:
        hr = max(2, radius // 3)
        draw.ellipse(
            [cx - hr, cy - hr, cx + hr, cy + hr],
            fill=(255, 255, 255),
        )


def _interp_peaks(peaks, x):
    """Linearly interpolate y value from peak list at given x."""
    for i in range(len(peaks) - 1):
        x0, y0 = peaks[i]
        x1, y1 = peaks[i + 1]
        if x0 <= x <= x1:
            t = (x - x0) / (x1 - x0) if x1 != x0 else 0
            return int(y0 + t * (y1 - y0))
    return peaks[-1][1]


# ---------------------------------------------------------------------------
# Play button: pixel-art style to replace the green GoMud button
# ---------------------------------------------------------------------------
def generate_play_button():
    W, H = 300, 100
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    border_color = (136, 136, 187)       # #8888bb (matches --button-shadow)
    bg_color = (59, 59, 120)             # #3b3b78 (matches --button-background)
    text_color = (208, 208, 232)         # #d0d0e8 (matches --text-primary-color)
    highlight_color = (160, 160, 208)    # #a0a0d0

    # Outer border (rounded-ish via nested rects)
    draw.rounded_rectangle([0, 0, W - 1, H - 1], radius=8,
                           outline=border_color, width=3)
    # Inner fill
    draw.rounded_rectangle([4, 4, W - 5, H - 5], radius=6, fill=bg_color)

    # Inner border highlight
    draw.rounded_rectangle([6, 6, W - 7, H - 7], radius=5,
                           outline=highlight_color, width=1)

    # "PLAY" text - pixel art style, drawn manually for crisp look
    # Each letter is roughly 40px wide, 40px tall, centered in the button
    # Total text width ~180px, centered at x=60..240
    lx, ly = 62, 28  # top-left of text block
    pw = 8   # pixel/stroke width
    lh = 44  # letter height

    # P
    x = lx
    draw.rectangle([x, ly, x + pw, ly + lh], fill=text_color)          # vertical
    draw.rectangle([x, ly, x + 28, ly + pw], fill=text_color)          # top bar
    draw.rectangle([x + 20, ly, x + 28, ly + pw * 3], fill=text_color) # right of top
    draw.rectangle([x, ly + pw * 2, x + 28, ly + pw * 3], fill=text_color)  # middle bar

    # L
    x = lx + 38
    draw.rectangle([x, ly, x + pw, ly + lh], fill=text_color)          # vertical
    draw.rectangle([x, ly + lh - pw, x + 28, ly + lh], fill=text_color)  # bottom bar

    # A
    x = lx + 76
    draw.rectangle([x, ly, x + pw, ly + lh], fill=text_color)          # left vert
    draw.rectangle([x + 20, ly, x + 28, ly + lh], fill=text_color)     # right vert
    draw.rectangle([x, ly, x + 28, ly + pw], fill=text_color)          # top bar
    draw.rectangle([x, ly + pw * 2, x + 28, ly + pw * 3], fill=text_color)  # middle bar

    # Y
    x = lx + 114
    draw.rectangle([x, ly, x + pw, ly + pw * 3], fill=text_color)      # left top
    draw.rectangle([x + 20, ly, x + 28, ly + pw * 3], fill=text_color) # right top
    draw.rectangle([x, ly + pw * 2, x + 28, ly + pw * 3], fill=text_color)  # middle
    draw.rectangle([x + 10, ly + pw * 2, x + 18, ly + lh], fill=text_color)  # center down

    out_path = os.path.join(IMAGES_DIR, "btn_play.png")
    img.save(out_path)
    print(f"  Wrote {out_path}")


# ---------------------------------------------------------------------------
if __name__ == "__main__":
    os.makedirs(IMAGES_DIR, exist_ok=True)
    print("Generating DOGMud art assets...")
    generate_favicon()
    generate_web_bg()
    generate_play_button()
    print("Done.")
