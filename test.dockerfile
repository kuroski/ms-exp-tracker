
# Use official Go image
FROM golang:1.24.2

LABEL maintainer="your-name@example.com"

# Install tesseract dependencies
RUN apt-get update -qq && apt-get install -y -qq \
    # 🧠 Wails dependencies (using webkit2gtk-4.0, not 4.1!)
    libwebkit2gtk-4.0-dev \
    libsoup2.4-dev \
    libgtk-3-dev \
    libglib2.0-dev \
    libgdk-pixbuf2.0-dev \
    libnotify-dev \
    libatk-bridge2.0-dev \
    libcairo2-dev \
    libx11-dev \
    libxtst-dev \
    build-essential \
    pkg-config \
    # 🔍 Tesseract OCR and gosseract dependencies
    libtesseract-dev \
    libleptonica-dev \
    tesseract-ocr \
    tesseract-ocr-eng \
    # Cleanup
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set TESSDATA_PREFIX in case needed by gosseract
ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/5/tessdata/

# RUN go install -v github.com/wailsapp/wails/v3/cmd/wails3@latest

# Set workdir and copy files
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Run your tests

CMD ["go", "test", "-tags", "wailswebview webkit2_4_0", "-v", "./..."]

