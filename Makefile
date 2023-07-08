TAG ?= latest

.PHONY: build
build:
	docker build -t keganhollern/aika:${TAG} .
	docker push keganhollern/aika:${TAG}

.PHONY: run
run: build
	docker run -e AIKA_DISCORD_KEY=${AIKA_DISCORD_KEY} -e OPENAI_KEY=${OPENAI_KEY} keganhollern/aika:${TAG} 