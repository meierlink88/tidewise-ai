FROM alpine:3.20

COPY run-evidence-receipt-cleanup.sh /control/run-evidence-receipt-cleanup.sh

ENTRYPOINT ["/bin/false"]
