# Debian slim rather than Alpine, matching stringer: musl gives up the
# better-trodden arm64 path, and the size it saves is nothing on three Pis
# with NVMe.
FROM node:24-bookworm-slim AS build

WORKDIR /app
COPY package.json package-lock.json tsconfig.json ./
RUN npm ci
COPY src ./src
COPY test ./test
RUN npm run build && npm prune --omit=dev

FROM node:24-bookworm-slim

WORKDIR /app
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/dist/src ./dist/src
# Migrations are read from disk at boot, so they ship with the image rather
# than being applied by a separate Job that Flux would have to reconcile.
COPY drizzle ./drizzle
COPY package.json ./

# Nothing here needs to write outside the spool volume, and nothing needs a name.
USER node
CMD ["node", "dist/src/index.js"]
