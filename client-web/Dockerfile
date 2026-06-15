FROM node:24-alpine AS build

WORKDIR /app
COPY client-web/package.json client-web/package-lock.json ./
RUN npm ci

COPY service-api/openapi.yaml /service-api/openapi.yaml
COPY client-web/ ./
RUN npm run test && npm run build

FROM nginx:1.29-alpine

COPY infra/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
