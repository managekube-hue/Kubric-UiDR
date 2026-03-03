# GHCR-focused build mapping for ndr-rita
FROM --platform=linux/amd64 ubuntu:24.04 AS helper

# Build with existing root Dockerfile:
# docker build --target ndr-rita -f Dockerfile.api -t ghcr.io/<owner>/<repo>/ndr-rita:<tag> .

CMD ["/bin/true"]
