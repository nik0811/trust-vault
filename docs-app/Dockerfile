FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source files
COPY . .

# Build the static export
RUN npm run build

# Production stage - use nginx to serve static files
FROM nginx:alpine

# Copy the static export to nginx
COPY --from=builder /app/out /usr/share/nginx/html

# Copy custom nginx config for SPA routing
RUN echo 'server { \
    listen 3002; \
    root /usr/share/nginx/html; \
    index index.html; \
    \
    location / { \
        try_files $uri $uri/ $uri.html /index.html; \
    } \
    \
    location /_next/static/ { \
        expires 1y; \
        add_header Cache-Control "public, immutable"; \
    } \
}' > /etc/nginx/conf.d/default.conf

EXPOSE 3002

CMD ["nginx", "-g", "daemon off;"]
