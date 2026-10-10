from fastapi import APIRouter, File, Form, UploadFile, HTTPException
from fastapi.responses import Response

from repos import ProcessRequest
from services import process_frames


router = APIRouter()


@router.post("/process")
async def process(
    meme: UploadFile = File(...),
    top_text: str = Form(""),
    bottom_text: str = Form(""),
    backdrop_url: str = Form(...),
    logo_url: str = Form(...),
    font_path: str | None = Form(None),
    font_size: int = Form(22),
):
    try:
        content = await meme.read()
        req = ProcessRequest(
            top_text=top_text,
            bottom_text=bottom_text,
            backdrop_url=backdrop_url,
            logo_url=logo_url,
            font_path=font_path,
            font_size=font_size,
        )
        data, media_type = process_frames(
            content,
            req.top_text,
            req.bottom_text,
            req.backdrop_url,
            req.logo_url,
            req.font_path,
            req.font_size,
        )
        return Response(content=data, media_type=media_type)
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


