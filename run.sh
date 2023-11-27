#!/bin/bash

docker run \
        -v ${PWD}/data:/aika/data \
        -e S3_ACCESS=${S3_ACCESS} \
        -e S3_SECRET=${S3_SECRET} \
        -e S3_BUCKET=${S3_BUCKET} \
        -e S3_HOSTNAME=${S3_HOSTNAME} \
        -e S3_REGION=${S3_REGION} \
        -e S3_PUBLICURL=${S3_PUBLICURL} \
        -e AIKA_DISCORD_KEY=${AIKA_DISCORD_KEY} \
        -e OPENAI_KEY=${OPENAI_KEY} \
        -e ELEVENLABS_APIKEY=${ELEVENLABS_APIKEY} \
        keganhollern/aika:$1