# Aika

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

Aika is a ChatGPT powered anime waifu for Discord. She is a companion, an assistant, and a utility.

## Desired Features

- text chat
- image gen (uncensored / stable diff)
  - some form of feedback to let the user know the image is being generated...
- code execution ([sandboxed python](https://github.com/cohere-ai/cohere-terrarium))
- voice chat (stremaing / low latency)
    - voice cloning (zero shot / "make your voice sound like X")
- chat "sessions"
    > basically, if the user doesn't message aika for a fixed period of time, the chat "times out", a summary of the chat can be fed in to subsequent message later on.
    >
    > so imagine messaging aika, having a convo, leaving for a day, and then you come back and say Hi. To aika this is a brand new chat, with her context maybe having a summary of the previous conversation with the user
- basic memory
    > "aika, from now on call me kgod" and she should always get this snippet of memory in her context window
- restart/crash persistence
- ubiquitous text/voice modes (chat and voice combined as one context / chat)
- web features 
- @ing / text markup 
- multiple frontends (discord, telegram, terminal, web)
- multimodal inputs and outputs in text
  - from voice "hey aika, generate an image of this" should go into chat
- token counting for chat length limits
- reminder / timed actions
  - "aika, remind me of X on Y"

