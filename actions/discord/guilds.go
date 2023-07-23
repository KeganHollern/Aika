package discord

import (
	"aika/discord/discordai"
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

type Guilds struct {
	Session *discordgo.Session
}

func (g *Guilds) GetFunction_ListGuilds() discordai.Function {
	return discordai.Function{
		Definition: definition_listGuilds,
		Handler:    g.handler_listGuilds,
	}
}

var definition_listGuilds = openai.FunctionDefinition{
	Name:        "listGuilds",
	Description: "Retrieve a list of all guilds aika is in.",

	Parameters: jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
		Required:   []string{},
	},
}

// handler for getRandomNumber
func (g *Guilds) handler_listGuilds(msgMap map[string]interface{}) (string, error) {

	obj, err := g.action_listGuilds()
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

type guildResponse struct {
	Guilds []guildEntry `json:"guilds"`
}
type guildEntry struct {
	Name string `json:"name"`
	Id   string `json:"Id"`
}

// raw function implementation
func (g *Guilds) action_listGuilds() (guildResponse, error) {
	res := guildResponse{
		Guilds: []guildEntry{},
	}
	for _, rootGd := range g.Session.State.Guilds {
		gd, err := g.Session.Guild(rootGd.ID)
		if err != nil {
			logrus.
				WithError(err).
				WithField("id", rootGd.ID).
				Warnln("could not find guild data")
			res.Guilds = append(res.Guilds, guildEntry{"unknown guild", gd.ID})
		} else {
			logrus.
				WithField("name", gd.Name).
				WithField("id", gd.ID).
				Info("in guild")

			res.Guilds = append(res.Guilds, guildEntry{gd.Name, gd.ID})
		}
	}
	return res, nil
}
