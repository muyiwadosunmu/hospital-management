
# Include variables from the .envrc file
include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #


## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
# Create the new confirm target.
.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #


## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	@echo 'Running Migrations'
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING=$(HOSPITAL_MGT_DSN) goose -dir $(GOOSE_MIGRATION_DIR) up

## db/migrations/down: revert database migrations
.PHONY: db/migrations/down
db/migrations/down:
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING=$(HOSPITAL_MGT_DSN) goose -dir $(GOOSE_MIGRATION_DIR) down

## db/migrations/reset: reset all database migrations
.PHONY: db/migrations/reset
db/migrations/reset:
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING=$(HOSPITAL_MGT_DSN) goose -dir $(GOOSE_MIGRATION_DIR) reset


## db/migrations/status: Check database migrations status
.PHONY: db/migrations/status
db/migrations/status:
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING=$(HOSPITAL_MGT_DSN) goose -dir $(GOOSE_MIGRATION_DIR) status

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/api -db-dsn=$(HOSPITAL_MGT_DSN)

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql $(HOSPITAL_MGT_DSN)

## db/migrations/new name=$1 dialect=$2: create a new database migration

.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	goose -dir ./migrations create ${name} ${dialect}


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #
## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...


## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor


# ==================================================================================== #
# BUILD
# ==================================================================================== #
current_time = $(shell date --iso-8601=seconds)
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.version=${git_description}'
## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/api ./cmd/api

# ==================================================================================== #
# PRODUCTION
# ==================================================================================== #
production_host_ip = "138.68.88.63"
## production/connect: connect to the production server
.PHONY: production/connect
production/connect:
	ssh greenlight@${production_host_ip}

## production/deploy/api: deploy the api to production
.PHONY: production/deploy/api
production/deploy/api:
	rsync -rP --delete ./bin/linux_amd64/api ./migrations greenlight@${production_host_ip}:~
	ssh -t greenlight@${production_host_ip} 'GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING=$(HOSPITAL_MGT_DSN) goose -dir $(GOOSE_MIGRATION_DIR) up'

## production/configure/api.service: configure the production systemd api.service file
.PHONY: production/configure/api.service

production/configure/api.service:
	rsync -P ./remote/production/api.service greenlight@${production_host_ip}:~
	ssh -t greenlight@${production_host_ip} '\
		sudo mv ~/api.service /etc/systemd/system/ \
		&& sudo systemctl enable api \
		&& sudo systemctl restart api'

## production/configure/caddyfile: configure the production Caddyfile
.PHONY: production/configure/caddyfile
production/configure/caddyfile:
	rsync -P ./remote/production/Caddyfile greenlight@${production_host_ip}:~
	ssh -t greenlight@${production_host_ip} '\
		sudo mv ~/Caddyfile /etc/caddy/ \
		&& sudo systemctl reload caddy'

# make migration name=test dialect=sql or go
# GOOSE_DRIVER=$GOOSE_DRIVER GOOSE_DBSTRING=$HOSPITAL_MGT_DSN goose -dir ./migrations up