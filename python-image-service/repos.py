from dataclasses import dataclass


@dataclass
class ProcessRequest:
    top_text: str
    bottom_text: str
    backdrop_url: str
    logo_url: str
    font_path: str | None
    font_size: int


