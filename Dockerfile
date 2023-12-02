FROM alpine:3.18.5
RUN apk update && apk upgrade && \
  apk add --no-cache ca-certificates
COPY stripe /bin/stripe
ENTRYPOINT ["/bin/stripe"]
