FROM scratch
COPY gdhk.linux.amd64 /gdhk

ENTRYPOINT ["/gdhk"]