cd database &&
docker build -t db .
docker run -p '14883:5432' --rm db