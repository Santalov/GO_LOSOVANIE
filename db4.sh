cd database &&
docker build -t db .
docker run -p '14884:5432' --rm db