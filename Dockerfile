FROM golang:1 AS build

WORKDIR /build
COPY . /build

RUN make restreamer

FROM scratch
LABEL maintainer="Gregor Riepl <onitake@gmail.com>"

COPY --from=build /build/restreamer /
COPY examples/minimal/restreamer.json /

EXPOSE 8000
ENTRYPOINT ["/restreamer"]
CMD ["/restreamer.json"]
