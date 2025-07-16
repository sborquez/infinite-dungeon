# Variables
SRC=app

run:
	cd $(SRC) && go run app -config config/local.yaml

build:
	cd $(SRC) && go build app
