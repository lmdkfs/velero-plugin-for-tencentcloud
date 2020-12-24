FROM golang:alpine
RUN mkdir /plugins
ADD _output/velero-* /plugins/
USER nobody:nobody
ENTRYPOINT ["/bin/bash", "-c", "cp /plugins/* /target/."]