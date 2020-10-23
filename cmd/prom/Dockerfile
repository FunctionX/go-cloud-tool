FROM prom/prometheus:latest

COPY ./build/prom /usr/local/bin/prom
COPY ./cmd/prom/start.sh start.sh

ENTRYPOINT ["./start.sh"]