.PHONY: docker
GIT_COMMIT=$(shell git rev-parse HEAD)

docker:
	docker build -t cinimex/alertmanager-megafon-sms:latest --build-arg GIT_COMMIT=$(GIT_COMMIT) .

