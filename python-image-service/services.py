import io
import os
from typing import Optional, Tuple, List

import requests
from PIL import Image, ImageDraw, ImageFont, ImageSequence

from constants import (
    FINAL_WIDTH,
    FINAL_HEIGHT,
    WATERMARK_SCALE,
    WATERMARK_MARGIN,
    TEXT_COLOR,
    STROKE_COLOR,
    STROKE_WIDTH,
    DEFAULT_FONT_CANDIDATES,
)


def load_image_from_url(url: str) -> Image.Image:
    r = requests.get(url, timeout=15)
    r.raise_for_status()
    return Image.open(io.BytesIO(r.content)).convert("RGBA")


def load_font(font_path: Optional[str], font_size: int) -> ImageFont.FreeTypeFont:
    candidates: List[str] = []
    if font_path:
        candidates.append(font_path)
    candidates.extend(DEFAULT_FONT_CANDIDATES)
    for p in candidates:
        if os.path.exists(p):
            try:
                return ImageFont.truetype(p, font_size)
            except Exception:
                continue
    return ImageFont.load_default()


def contain(frame: Image.Image, target_w: int, target_h: int) -> Image.Image:
    w, h = frame.size
    scale = min(target_w / w, target_h / h)
    new_w, new_h = int(w * scale), int(h * scale)
    return frame.resize((new_w, new_h), Image.LANCZOS)


def draw_text_with_stroke(draw: ImageDraw.ImageDraw, xy, text: str, font: ImageFont.ImageFont):
    draw.text(xy, text, font=font, fill=TEXT_COLOR, stroke_fill=STROKE_COLOR, stroke_width=STROKE_WIDTH)


def process_frames(content: bytes, top_text: str, bottom_text: str, backdrop_url: str, logo_url: str, font_path: Optional[str], font_size: int) -> Tuple[bytes, str]:
    original = Image.open(io.BytesIO(content))
    backdrop = load_image_from_url(backdrop_url)
    logo = load_image_from_url(logo_url)
    font = load_font(font_path, font_size)
    frames: List[Image.Image] = []
    durations: List[int] = []
    frame_count = getattr(original, "n_frames", 1)
    for i, frame in enumerate(ImageSequence.Iterator(original)):
        durations.append(frame.info.get("duration", 100))
        base = Image.new("RGBA", (FINAL_WIDTH, FINAL_HEIGHT))
        base.alpha_composite(backdrop.resize((FINAL_WIDTH, FINAL_HEIGHT), Image.LANCZOS))
        contained = contain(frame.convert("RGBA"), FINAL_WIDTH, FINAL_HEIGHT)
        ox = (FINAL_WIDTH - contained.size[0]) // 2
        oy = (FINAL_HEIGHT - contained.size[1]) // 2
        base.alpha_composite(contained, (ox, oy))
        draw = ImageDraw.Draw(base)
        if top_text:
            tw, th = draw.textbbox((0, 0), top_text, font=font)[2:]
            tx = (FINAL_WIDTH - tw) / 2
            ty = oy + 10
            draw_text_with_stroke(draw, (tx, ty), top_text, font)
        if bottom_text:
            bw, bh = draw.textbbox((0, 0), bottom_text, font=font)[2:]
            bx = (FINAL_WIDTH - bw) / 2
            by = oy + contained.size[1] - bh - 10
            draw_text_with_stroke(draw, (bx, by), bottom_text, font)
        wm_w = int(FINAL_WIDTH * WATERMARK_SCALE)
        wm_h = int(wm_w * (logo.height / logo.width if logo.width else 1))
        base.alpha_composite(logo.resize((wm_w, wm_h), Image.LANCZOS), dest=(FINAL_WIDTH - wm_w - WATERMARK_MARGIN, FINAL_HEIGHT - wm_h - WATERMARK_MARGIN))
        frames.append(base.convert("P", palette=Image.ADAPTIVE, colors=256, dither=Image.NONE))
        if i + 1 >= frame_count:
            break
    out = io.BytesIO()
    frames[0].save(out, format="GIF", save_all=True, append_images=frames[1:], duration=durations, loop=0, disposal=2, optimize=True)
    out.seek(0)
    return out.read(), "image/gif"


