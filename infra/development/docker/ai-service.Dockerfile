FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates 
    
ADD shared shared
ADD build build

ENTRYPOINT build/ai-service
