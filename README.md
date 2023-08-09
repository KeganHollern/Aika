# Aika

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?style=for-the-badge&logo=discord&logoColor=white)
![ChatGPT](https://img.shields.io/badge/chatGPT-74aa9c?style=for-the-badge&logo=openai&logoColor=white)
![YouTube](https://img.shields.io/badge/YouTube-%23FF0000.svg?style=for-the-badge&logo=YouTube&logoColor=white)
![DuckDuckGo](https://img.shields.io/badge/DuckDuckGo-DE5833?style=for-the-badge&logo=DuckDuckGo&logoColor=white)

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
$ ./run.sh beta
```

*see [run.sh](./run.sh) for environment variables required.*

## TODO

Add more admin commands
Improve logging
Token counting rather than history limit?
Drop history after X hours? How can we be cost effective?
"Reminder" function -- ask aika what she wants added
Let aika pull photos of 'herself' from S3
Improve youtube download