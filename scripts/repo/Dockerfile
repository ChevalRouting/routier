ARG ALPINE_IMAGE=docker.io/library/alpine:3.23
FROM $ALPINE_IMAGE

RUN apk add --no-cache abuild openssl python3 tar

ENTRYPOINT ["python3", "/repo/repo.py"]
