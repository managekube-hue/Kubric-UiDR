# GHCR-focused build mapping for coresec
FROM --platform=linux/amd64 ubuntu:24.04 AS helper

# This file documents and standardizes the GHCR build path.
# Actual build command should use the existing root multi-stage Dockerfile:
# docker build --target coresec -f Dockerfile.agents -t ghcr.io/<owner>/<repo>/coresec:<tag> .

CMD ["/bin/true"]
