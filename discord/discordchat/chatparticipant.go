package discordchat

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

type ChatParticipant struct {
	User *discordgo.User
}

func (p *ChatParticipant) GetMentionString() string {
	return fmt.Sprintf("<@%s>", p.User.ID)
}

func (p *ChatParticipant) GetDisplayName() string {
	// This regular expression matches any character that is not a letter or a number
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	processedString := reg.ReplaceAllString(p.User.Username, "")
	return processedString
}
