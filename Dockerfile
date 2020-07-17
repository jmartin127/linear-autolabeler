############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
# Need CA certs for calling Linear over HTTPS
# Need tzdata to get timezones
RUN apk update && apk add --no-cache git ca-certificates tzdata

# Create appuser
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/jmartin127/linear-autolabeler/
COPY . .
# Fetch dependencies.
# Using go get.
RUN go get -d -v
# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags '-w -s' -a -installsuffix cgo -o /go/bin/linear-autolabeler

############################
# STEP 2 build a small image
############################
FROM scratch

# Import the user and group files from the builder.
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable.
COPY --from=builder /go/bin/linear-autolabeler /go/bin/linear-autolabeler

# Copy CA certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy Timezone info
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Use an unprivileged user.
USER appuser:appuser

# Run the binary.
ENV token defaultToken
ENTRYPOINT ["/go/bin/linear-autolabeler"]

