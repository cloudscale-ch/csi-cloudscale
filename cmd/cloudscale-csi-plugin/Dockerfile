FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git make bash

WORKDIR /src

# Copy go.mod/go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown
ARG GIT_TREE_STATE=unknown

# Build using make (ensures consistent build logic)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make compile \
    VERSION="${VERSION}" \
    COMMIT="${COMMIT}" \
    GIT_TREE_STATE="${GIT_TREE_STATE}"

FROM alpine:3.23.3

# e2fsprogs-extra is required for resize2fs used for the resize operation
# blkid: block device identification tool from util-linux
RUN apk add --no-cache ca-certificates \
    e2fsprogs \
    findmnt \
    xfsprogs \
    cryptsetup \
    udev \
    blkid \
    e2fsprogs-extra

COPY --from=builder /src/cmd/cloudscale-csi-plugin/cloudscale-csi-plugin /bin/
COPY --from=builder /src/cmd/cloudscale-csi-plugin/csi-diskinfo.sh /bin/

ENTRYPOINT ["/bin/cloudscale-csi-plugin"]
