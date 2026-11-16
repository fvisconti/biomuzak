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

# Pre-download the MAEST model (discogs-maest-10s-pw-2, v2) at build time so
# the service needs no network access at runtime and every image is
# reproducible. The sha256 pins the exact artifact (a changed .pb would
# silently change every embedding the cosine index is built over); the
# location is overridable via MAEST_MODEL_PATH. ~344 MB.
RUN mkdir -p /models \
	&& curl -fsSL --retry 3 -o /models/discogs-maest-10s-pw-2.pb \
		https://essentia.upf.edu/models/feature-extractors/maest/discogs-maest-10s-pw-2.pb \
	&& echo "e0b2701cc35d89e9632a0948523aa70629ecbea98eef8d90d30ea0d8ae9d7ce3  /models/discogs-maest-10s-pw-2.pb" \
		| sha256sum -c -
ENV MAEST_MODEL_PATH=/models/discogs-maest-10s-pw-2.pb

# Copy the rest of the source code
COPY . .

EXPOSE 8000

CMD ["poetry", "run", "uvicorn", "audio_processor.main:app", "--host", "0.0.0.0", "--port", "8000"]
