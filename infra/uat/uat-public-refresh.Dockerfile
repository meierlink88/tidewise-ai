FROM postgres:16.14-alpine@sha256:57c72fd2a128e416c7fcc499958864df5301e940bca0a56f58fddf30ffc07777

RUN apk add --no-cache openssl

COPY restore-public-schema.sh /usr/local/bin/restore-public-schema
COPY snapshot.dump.enc /snapshot/snapshot.dump.enc

RUN chmod 0555 /usr/local/bin/restore-public-schema \
    && chmod 0444 /snapshot/snapshot.dump.enc

USER postgres

ENTRYPOINT ["/usr/local/bin/restore-public-schema"]
