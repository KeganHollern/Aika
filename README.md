# Aika
[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com) [![forthebadge](https://forthebadge.com/images/badges/kinda-sfw.svg)](https://forthebadge.com) [![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](https://forthebadge.com)

ChatGPT powered waifu.

![Aika Kissing](./assets/example.png)

## User Features

- ChatGPT
- DallE
- Web Search
- Waifu Image Gen
- Random Number Gen
- MyAnimeList Search
- Can @ chat members

## Developer Notes

- Admin Commands
- S3 DallE Proxy
- Dockerized for easy distribution
- Guild and DM specific GPT version control
- Guild and DM specific function control

## Run

```shell
$ ./run.sh latest
```

*see [run.sh](./run.sh) for environment variables required.*

## TODO

Add more admin commands
Improve logging
Implement configurable character message
Token counting rather than history limit?
Drop history after X hours? How can we be cost effective?
"Reminder" function -- ask aika what she wants added