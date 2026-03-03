# GHCR-focused build mapping for netguard
FROM --platform=linux/amd64 ubuntu:24.04 AS helper

# Build with existing root Dockerfile:
# docker build --target netguard -f Dockerfile.agents -t ghcr.io/<owner>/<repo>/netguard:<tag> .

CMD ["/bin/true"]
