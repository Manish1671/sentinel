# syntax=docker/dockerfile:1

FROM python:3.12-slim-bookworm
WORKDIR /app
RUN useradd -r -u 65532 -d /app sentinel
COPY services/ai/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt
COPY services/ai /app
COPY database/migrations /app/database/migrations
COPY evaluation /app/evaluation
ENV PYTHONPATH=/app
ENV MIGRATIONS_PATH=/app/database/migrations
USER 65532:65532
EXPOSE 8000
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
