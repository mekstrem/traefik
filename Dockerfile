FROM scratch
COPY script/ca-certificates.crt /etc/ssl/certs/
COPY dist/traefik /
LABEL foo=bar
EXPOSE 80
VOLUME ["/tmp"]
ENTRYPOINT ["/traefik"]
