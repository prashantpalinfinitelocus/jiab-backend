from fastapi import FastAPI
import routes

app = FastAPI(title="Python Image Processor", version="1.0.0")
app.include_router(routes.api)


