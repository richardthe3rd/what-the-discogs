FROM scratch
COPY wtd /wtd
ENTRYPOINT ["/wtd", "mcp"]
