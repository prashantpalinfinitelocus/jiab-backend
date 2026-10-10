FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Install curl for healthchecks
RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py ./
COPY constants.py ./
COPY services.py ./
COPY repos.py ./
COPY controllers.py ./
COPY routes.py ./

EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["sh","-c","curl -sf http://127.0.0.1:8081/healthz || exit 1"]
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8081"]


