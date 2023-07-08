package actions

import "github.com/bwmarrin/discordgo"

type Discord struct {
	Session *discordgo.Session
}

// TODO: this is kinda fucked and the messages aika calls do not have context (the discord session)
// because of that the whole concept of something like a "action_GetMembers()" is fucked
