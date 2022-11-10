FROM golang:1.18.8-buster

# Install deps
RUN apt-get update && apt-get install -y \
  libssl-dev \
  ca-certificates \
  fuse

ENV KUBO_SRC_DIR /kubo
ENV KUBO_CM_SRC_DIR /kubo-car-mirror

# Download packages first so they can be cached.
COPY kubo-car-mirror/go.mod kubo-car-mirror/go.sum $KUBO_CM_SRC_DIR/
RUN cd $KUBO_CM_SRC_DIR \
  && go mod download

COPY kubo/go.mod kubo/go.sum $KUBO_SRC_DIR/
RUN cd $KUBO_SRC_DIR \
  && go mod download

COPY kubo $KUBO_SRC_DIR
COPY kubo-car-mirror $KUBO_CM_SRC_DIR

RUN cd $KUBO_CM_SRC_DIR \
  && make build

# TODO...
