FROM postgres:latest


ENV POSTGRES_USER=admin
ENV POSTGRES_PASSWORD=1
ENV POSTGRES_DB=jqk

RUN mkdir -p /var/lib/postgresql/data && chown -R postgres:postgres /var/lib/postgresql

EXPOSE 5432
