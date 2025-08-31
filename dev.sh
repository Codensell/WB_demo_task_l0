docker compose down

docker build --tag go_orders_backend:latest .

docker compose up -d

docker exec broker /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --create --if-not-exists --topic orders \
    --replication-factor 1 --partitions 1