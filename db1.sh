cd database &&
docker build -t db .
docker run -p '14881:5432' --rm db