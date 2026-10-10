from fastapi import APIRouter

import controllers


api = APIRouter()
api.include_router(controllers.router)


