docker compose down

docker compose up -d

# docker exec kafka kafka-topics.sh --bootstrap-server localhost:9092 \
#       --create --if-not-exists --topic orders \
#       --replication-factor 1 --partitions 1