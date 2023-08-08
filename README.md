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
- Youtube Search
- Youtube Video Download

## Developer Notes

- Admin Commands
- DallE generations go to S3 storage
- Dockerized for easy distribution
- Guild and DM specific GPT version control
- Guild and DM specific function control
- Youtube 'downloads' go to S3 storage

## Run

```shell
$ ./run.sh latest
```

*see [run.sh](./run.sh) for environment variables required.*

## TODO

Add more admin commands
Improve logging
Token counting rather than history limit?
Drop history after X hours? How can we be cost effective?
"Reminder" function -- ask aika what she wants added
Let aika pull photos of 'herself' from S3