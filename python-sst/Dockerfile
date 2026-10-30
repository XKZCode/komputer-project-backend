FROM python:3.10-slim

# Install dependency sistem untuk ffmpeg, git, dan portaudio
RUN apt-get update && apt-get install -y \
    ffmpeg \
    git \
    portaudio19-dev \
    && rm -rf /var/lib/apt-get/lists/*

WORKDIR /app

# Copy kebutuhan pip dan install
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]