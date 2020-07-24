cd database &&
docker build -t db .
docker run -p '14885:5432' --rm db