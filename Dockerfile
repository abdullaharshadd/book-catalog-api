FROM node:18-slim

# Install OpenSSL 1.1 and other dependencies
RUN apt-get update -y && \
    apt-get install -y --no-install-recommends \
        openssl \
        libssl-dev \
        ca-certificates \
        wget \
    && wget http://archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb 2>/dev/null || true \
    && dpkg -i libssl1.1_1.1.1f-1ubuntu2_amd64.deb 2>/dev/null || true \
    && rm -f libssl1.1_1.1.1f-1ubuntu2_amd64.deb \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package*.json ./
COPY tsconfig*.json ./
COPY prisma ./prisma/

RUN npm install

COPY src ./src/

RUN npx prisma generate --schema=prisma/schema.prisma || true

RUN npm run build || npx tsc || true

EXPOSE 3000

CMD ["node", "dist/index.js"]