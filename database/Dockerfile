FROM postgres:12

ENV POSTGRES_USER=blockchain
ENV POSTGRES_PASSWORD=ffff
ENV POSTGRES_DB=blockchain
ENV POSTGRES_HOST_AUTH_METHOD=trust

COPY ./initdb/* /docker-entrypoint-initdb.d/
EXPOSE 5432