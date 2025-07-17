# Variables
SRC=app

tidy:
	cd $(SRC) && go mod tidy

run:
	cd $(SRC) && go run app -config config/local.yaml

build:
	cd $(SRC) && go build app
