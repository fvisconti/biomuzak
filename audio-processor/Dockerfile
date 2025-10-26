FROM python:3.11-slim

ENV POETRY_VERSION=2.0.1 \
	POETRY_VIRTUALENVS_CREATE=false \
	PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
		build-essential \
		cmake \
		pkg-config \
		curl \
		ca-certificates \
		git \
		libfftw3-dev \
		libsamplerate0-dev \
		libyaml-dev \
		libtag1-dev \
		libavcodec-dev \
		libavformat-dev \
		libavutil-dev \
		zlib1g-dev \
		libchromaprint-dev \
		libeigen3-dev \
	&& rm -rf /var/lib/apt/lists/*

# Install Poetry
RUN pip install "poetry==${POETRY_VERSION}"

# Copy dependency definitions first for better Docker caching
COPY pyproject.toml ./
# If you later add a poetry.lock it will be used for deterministic builds
# COPY poetry.lock ./

# Install dependencies (no project code yet)
RUN poetry install --no-interaction --no-ansi --only main --no-root

# Copy the rest of the source code
COPY . .

EXPOSE 8000

CMD ["poetry", "run", "uvicorn", "audio_processor.main:app", "--host", "0.0.0.0", "--port", "8000"]
