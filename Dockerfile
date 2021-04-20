FROM golang:alpine as builder

ADD . /usr/src/cni-ipoib-st

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

RUN apk add --update --virtual build-dependencies build-base linux-headers git
RUN cd /usr/src/cni-ipoib-st && \
    go env -w GOPROXY=https://goproxy.cn,direct && \
    make clean && \
    make build


FROM alpine
COPY --from=builder /usr/src/cni-ipoib-st/build/cni-ipoib-ts /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="CNI-IPOIB-TS"

ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
