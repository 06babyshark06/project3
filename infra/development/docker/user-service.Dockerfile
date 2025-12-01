FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y \
    librdkafka1 \
    ca-certificates 
    
ADD shared shared
ADD build build

ENTRYPOINT build/user-service