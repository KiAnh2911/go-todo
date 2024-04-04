docker_go:
	docker run -d --name demo-mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=todo_db mysql:8.0

dev:
	go run main.go
# make new_migration MESSAGE_NAME="message name"
new_migration:
	migrate create -ext sql -dir scripts/migration/ -seq $(MESSAGE_NAME)
# run db migration (change your connection string if your config differs)
up_migration:
	migrate -path scripts/migration/ -database "mysql://root:123456@tcp(127.0.0.1:3306)/todo_db?charset=utf8mb4&parseTime=True&loc=Local" -verbose up

down_migration:
	migrate -path scripts/migration/ -database "mysql://root:123456@tcp(127.0.0.1:3306)/todo_db?charset=utf8mb4&parseTime=True&loc=Local" -verbose down

# remove docker container
docker_container: 
	docker rm -f $(docker ps -aq)

# remove docker volumes & images
docker_clear:
	docker volume rm -f $(docker volume ls -qf "dangling=true" -q) & docker rmi --force $(docker images -f "dangling=true" -q)

# up all services with forced  build and re-create options
compose_up_rebuild:
	docker compose up --build --force-recreate

# docker compose up all services without rebuilding images
compose_up:
	docker compose up

# docker compose down
compose_down:
	docker compose down