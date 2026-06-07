# Stage 1: Build the Vue application
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY package.json package-lock.json ./
RUN npm ci

# Copy frontend source files
COPY . .

# Build application for production
RUN npm run build

# Stage 2: Serve static files with Nginx
FROM nginx:stable-alpine

# Copy built HTML/JS assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy routing configuration for SPA
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 82

CMD ["nginx", "-g", "daemon off;"]
