# syntax=docker/dockerfile:1

FROM node:20-alpine AS deps
WORKDIR /app
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci

FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY apps/web ./
ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_OUTPUT=standalone
RUN npx next build

FROM node:20-alpine AS runner
WORKDIR /app
RUN addgroup -g 65532 -S sentinel && adduser -S -u 65532 -G sentinel sentinel
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
COPY --from=build --chown=65532:65532 /app/public ./public
COPY --from=build --chown=65532:65532 /app/.next/standalone ./
COPY --from=build --chown=65532:65532 /app/.next/static ./.next/static
USER 65532:65532
EXPOSE 3000
CMD ["node", "server.js"]
