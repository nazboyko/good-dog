# One binary, one machine, one file to roll back.
#
# The frontend is built first and embedded into the Go binary, so the
# deployed artifact has no static file server, no CDN and no second
# origin to get out of sync with.

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --silent
COPY web/ ./
RUN npm run build

# Must be at least the version in go.mod, which is the one place the
# toolchain is declared. CI reads it with go-version-file; this line is
# the only copy of it in the repo and the only one that can drift.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# the frontend goes where webui embeds it, and the build fails loudly
# rather than shipping a binary that serves an empty page
COPY --from=web /src/web/dist ./internal/webui/dist
RUN test -f internal/webui/dist/index.html || (echo "frontend missing from the image" && exit 1)
# CGO off because modernc.org/sqlite is pure Go: that is the whole
# reason it was chosen, see docs/tech-stack.md
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/good-dog ./cmd/server

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /out/good-dog /app/good-dog

# The recorded radio ships in the image rather than living on the
# volume. It is 4MB, it is deterministic, and it means a judge hears the
# night on a machine that has never run `make voices`. The volume is for
# what changes: the sqlite file.
COPY cache/audio /app/cache/audio
COPY fixtures /app/fixtures
COPY cache/sheets /app/cache/sheets
COPY cache/photos /app/cache/photos
COPY assets /app/assets

ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/good-dog"]
