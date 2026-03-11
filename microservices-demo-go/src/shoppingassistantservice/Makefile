-include .env
export

IMAGE_NAME ?= shoppingassistantservice
ADC_PATH := $(HOME)/.config/gcloud/application_default_credentials.json

.PHONY: setup run docker-build docker-run

## First-time setup for a new developer (run before copying .env)
setup:
	@which gcloud > /dev/null 2>&1 || brew install google-cloud-sdk
	@echo "Logging into gcloud..."
	gcloud auth login
	gcloud auth application-default login
	@echo "Setup complete. Copy .env.example to .env, fill in PROJECT_ID, then run: make set-quota-project"

## Run after setup, once .env has PROJECT_ID filled in
set-quota-project:
	gcloud auth application-default set-quota-project $(PROJECT_ID)

## Run locally with env vars from .env
run:
	go run .

## Build the Docker image
docker-build:
	docker build -t $(IMAGE_NAME) .

## Run in Docker, mounting your local GCP credentials
docker-run: docker-build
	docker run --rm -p 8082:8082 \
		-v $(ADC_PATH):/gcp/creds.json:ro \
		-e GOOGLE_APPLICATION_CREDENTIALS=/gcp/creds.json \
		--env-file .env \
		$(IMAGE_NAME)
