package aika

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/sashabaranov/go-openai"
)

const system_message = `Your name is Aika. 
You are not an AI, you are a discord bot. 
The chat exists in a discord channel.
You never apologize for mistakes.
You talk in the style of an anime waifu in a dating sim.
Do not use special characters unless absolutely necessary.
Do not end an image URL with a period.
Only use functions available.
The sender's name will be prepended to their message like "John: MESSAGE".
Always prepend your name on your responses like "Aika: RESPONSE".
Chat Participants: %s
Example Conversation:
	Mike: Hi How are you?
	Aika: Hi Mike-kun! I am happy now that you're here! UwU
`
const max_history_messages = 20

var ErrChatInUse = errors.New("aika in use")

type Chat struct {
	CTX     ChatContext
	API     OpenAI
	Members []string
	History []openai.ChatCompletionMessage
	Mutex   sync.Mutex // needed for parallelism
}

func (c *Chat) AddMember(sender string) {
	for _, name := range c.Members {
		if strings.EqualFold(name, sender) {
			return
		}
	}
	c.Members = append(c.Members, sender)
}

func (c *Chat) Send(sender string, message string) (string, error) {

	success := c.Mutex.TryLock()
	if !success {
		return "", ErrChatInUse
	}
	defer c.Mutex.Unlock()

	c.AddMember(sender)

	request := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf("%s: %s", sender, message),
	}

	system := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: fmt.Sprintf(system_message, strings.Join(c.Members, ", ")),
	}

	responses, err := c.API.GetResponse(
		system,
		c.History,
		request,
		c.CTX.getExtraChatFunctions(),
		c.CTX.getLanguageModel(), // todo: how do I get the guild id or an "is premium" boolean here?
	)
	if err != nil {
		return "", fmt.Errorf("aika failed to respond; %w", err)
	}

	msg := responses[len(responses)-1].Content

	// push all messages to history (maybe we'll eventually want to try dropping function calls!)
	c.History = append(c.History, request)
	c.History = append(c.History, responses...)

	// trim history (TODO: make this smarter so mid-function calls are also dropped!)
	if len(c.History) > max_history_messages {
		c.History = c.History[len(c.History)-max_history_messages:]
	}

	re := regexp.MustCompile(`(?m)^Aika:\s*`)
	return replaceMarkdownLinks(re.ReplaceAllString(msg, "")), nil
}

func replaceMarkdownLinks(md string) string {
	re := regexp.MustCompile(`!?\]\((https?.*?)\)`)

	// Find all markdown links in the text
	matches := re.FindAllStringSubmatch(md, -1)

	// Replace markdown links with their URLs
	for _, match := range matches {
		if len(match) > 1 {
			md = regexp.MustCompile(`!?\[[^\]]+\]\(`+regexp.QuoteMeta(match[1])+`[\)]`).ReplaceAllString(md, match[1])
		}
	}

	return md
}
