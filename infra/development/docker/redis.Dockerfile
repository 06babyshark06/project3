# Dockerfile
FROM redis:latest

RUN mkdir -p /data && chown -R redis:redis /data

EXPOSE 6379

CMD ["redis-server"]
