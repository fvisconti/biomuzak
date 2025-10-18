# Use an official Python runtime as a parent image
FROM python:3.12-slim

# Set the working directory in the container
WORKDIR /app

# Install uv
RUN pip install uv

# Copy the pyproject.toml file into the container at /app
COPY pyproject.toml .

# Install any needed packages specified in pyproject.toml
RUN uv pip install --system --no-cache-dir .

# Copy the rest of the application's code
COPY . .

# Expose the port the app runs on
EXPOSE 8000

# Run the application
CMD ["uvicorn", "audio_processor.main:app", "--host", "0.0.0.0", "--port", "8000"]
