# Aika

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?style=for-the-badge&logo=discord&logoColor=white)
![ChatGPT](https://img.shields.io/badge/chatGPT-74aa9c?style=for-the-badge&logo=openai&logoColor=white)
![YouTube](https://img.shields.io/badge/YouTube-%23FF0000.svg?style=for-the-badge&logo=YouTube&logoColor=white)
![DuckDuckGo](https://img.shields.io/badge/DuckDuckGo-DE5833?style=for-the-badge&logo=DuckDuckGo&logoColor=white)

Aika is a ChatGPT powered anime waifu for Discord. She is a companion, an assistant, and a utility.

<img width="400px" src="https://aika.lystic.zip/Screenshot%202023-11-20%20at%207.00.40%20PM.png"/>


## Features

Aika is more than just a fun chat bot. She has functional integrations with many services and can chain these together to assist users in nearly any task. 

Here is an exhaustive list of what she can do:
- Text-based interaction via [ChatGPT](https://platform.openai.com/docs/guides/text-generation/chat-completions-api)
- Image generation via [Dall-E](https://platform.openai.com/docs/guides/images/introduction)
- Basic web searching via [DuckDuckGo](https://duckduckgo.com/)
- Acquire human-created waifu images via [Waifu.pics](https://waifu.pics/)
- Random number generation
- Anime lookup via [MyAnimeList](https://myanimelist.net/)
- Tag individual members in her messages (@ing)
- Search [YouTube](https://www.youtube.com/) for videos
- Download [YouTube](https://www.youtube.com/) videos to MP4
- **Join voice chat and speak**
- Play music in voice chat via [YouTube](https://www.youtube.com/)

Aika can chain any of these actions together as commanded and at will. For example:
> You: "Aika join voice and play Never Gonna Give You Up."\
> Aika: *joins voice chat and starts trolling.* "Okay if you say so..."

You can always ask Aika what functions she can use:
> You: "Aika what functions can you use?"\
> Aika: "I can ...."

She incorperates information acquired into her responses:
> You: "Aika what is the weather in the bay?"\
> Aika: *searches the web.* "The weather in the Bay Area is 70 degress and sunny."

She loves tagging people in her messages:
> You: "Aika flip a coin. If it's heads tell everyone we're playing poker tonight."\
> Aika: *random number 0-1: 1.* "Sorry @Kegan, but no one is playing poker tonight."

## Voice Chat

She has a voice! You can talk to her. I talk to her every day... ([Click Here to Listen](https://aika.lystic.zip/user-content/sample_clip.mp3))

While in voice chat, she has all the same functionality as in text chat.
Her average response time is **less than 2 seconds**. 

To talk to her:
1. Join a voice channel she has permissions to.
2. Ask aika to join the voice chat.
3. Wait ~2 seconds after she joins.
4. Start talking to her!

Aika requires the keyword `Aika` be spoken in order to respond.
> "Hey **Aika**, How Are you?"

When she is responding to **you**, you can continue the conversation naturally by replying back to her within **5 seconds**.
> You: "Hi **Aika**!"\
> Aika: "Hi..."\
> You: "How are you?"\
> Aika: "Good...."

Aika uses voice activity. Hotmicing, or having an open microphone with background noise, causes a lot of problems with communication.
I recommend using Krisp to silence any background noise.

## Vision

Aika has vision integration. She can see images you share with her & interact with them!

Image details are embedded in Aika's context window. This means Aika can use image details throughout the conversation.

<img width="800px" src="https://aika.lystic.zip/cool_image.jpg"/>

Images can be attached or linked to. When linking to images, a direct link _must_ be used for all services _except_ for Tenor.
Discord's built in GIF selection works with Aika.

## Self Hosting

Running your own Aika-themed bot can be easy. 

1. Updated the [system messages](./discord/discordai/) with your own persona-themed ones.
2. [Build](#build)
3. [Run](#run)

### Dependencies / APIs

Aika requires a few depenedencies to operate.

1. An [S3](https://aws.amazon.com/s3/) compatible object store
2. A [Discord Bot](https://discord.com/developers/docs/getting-started) API key
3. An [OpenAI](https://platform.openai.com/docs/quickstart/account-setup) API key
4. An [ElevenLabs](https://elevenlabs.io/) API key

### Build

Aika is containerized via [Docker](https://www.docker.com/). No special build requirements are needed, simply use `docker build`

```shell
$ docker build -t mycustom/dockertag
```

### Run

1. Create a data folder
```shell
$ mkdir data
```
2. Copy the [config](./data/config.yaml) into that data folder. Edit as needed.
3. Set the required environment variables.\
*see [run.sh](./run.sh) for environment variables.*
4. Run via the run script.\
*if building a custom iamge, update the script with your own image.*
```shell
$ ./run.sh beta
```

## TODO

Voice Chat & Audio Mixer Refactor (hacked in right now)

Add more guild & operator admin commands
- let aika control guild as admin bot for guild owners?
- let operator enable and disable "premium" guilds via chat
- let operator overwrite system message at runtime
- let operator force aika out of discords

Improve guild configurations
- "premium or not" is not granular enough

Further imrpovements to voice chat for natural interaction

Token counting rather than history limit
- will increase costs
- will improve bot context
- will fix issues around bots generating essays and such

Drop history after X hours of inactivity / cost efficiency?
- if someone doesn't message aika for 24hrs they're probably starting a new chat

Report/Track token usage by guild/user
- will be needed for cost / expense tracking
- rate limiting potential on a per-user basis

"Reminder / Alert" function so Aika can DM users @ specific times for specific things
- unsure how to get aika to understand she's responding to a reminder and not a real human message

"Let aika pull photos of 'herself' from S3
- i have several profile pics of her so this shud be easy hardcoded thing configurable

Improve youtube download for cost efficiency
- smol videos bcz this feature is niche

Reduce voice interaction latency further
- faster TTS
- any TTS API that takes text streaming?
- better TTS apis?
- faster tok/s from GPT (4-turbo is nice)
- can voice use 3.5-turbo if its faster?

Investigate alternative transcription APIs
- whisper is just OK

Investigate alternative TTS APIs (Like OAI and PlayHT)
- Looking for faster response times & higher quality

operator-controlled runtime voice cloning
- "aika sound like X" -> she clones Xs voice and starts using it immediately

Aika currently leaves voice chat without speaking, maybe fix this?
- would require some weird hacky action integration w/ voice chats to prevent the action from actually running until after she finishes speaking. Something like a "run after replying" ability ?

Dall-E 3 is slow, can we tell the user when Aika is waiting on it?
- either a progress bar... or like some way of letting the user know she didn't freeze up




