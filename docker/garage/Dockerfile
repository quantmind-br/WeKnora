# Garage (S3-compatible storage) server + init image.
# Wraps the upstream garage binary on debian-slim so entrypoint/init scripts
# (sh, sed, curl) can render garage.toml from env and run one-shot cluster setup.
FROM dxflrs/garage:v2.1.0 AS upstream

FROM debian:stable-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl \
  && rm -rf /var/lib/apt/lists/*

COPY --from=upstream /garage /usr/local/bin/garage
COPY docker/garage/entrypoint.sh docker/garage/init.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/init.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
