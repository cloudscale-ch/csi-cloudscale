# Copyright cloudscale.ch
# Copyright 2018 DigitalOcean
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
FROM golang:1.27.1 AS builder

WORKDIR /src

# Copy go.mod/go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all necessary source code
COPY Makefile ./
COPY driver/ driver/
COPY cmd/ cmd/

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown
ARG GIT_TREE_STATE=unknown

# Build using make (ensures consistent build logic)
RUN make compile \
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
