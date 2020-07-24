cd database &&
docker build -t db .
docker run -p '14882:5432' --rm db