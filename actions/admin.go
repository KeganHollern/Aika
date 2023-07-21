package actions

import (
	"aika/premium"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

// because no struct is necessary for math functions
// they can be inline defined

type Admin struct {
	PremiumDB *premium.Servers
}

func (adm *Admin) GetFunction_AddPremiumGuild() Function {
	return Function{
		Definition: definition_addPremiumGuild,
		Handler:    adm.handler_AddPremiumGuild,
	}
}
func (adm *Admin) GetFunction_RemovePremiumGuild() Function {
	return Function{
		Definition: definition_removePremiumGuild,
		Handler:    adm.handler_RemovePremiumGuild,
	}
}
func (adm *Admin) GetFunction_GetPremiumGuilds() Function {
	return Function{
		Definition: definition_getPremiumGuilds,
		Handler:    adm.handler_GetPremiumGuilds,
	}
}

var (
	errGuildAlreadyPremium = errors.New("guild already premium")
	errGuildNotPremium     = errors.New("guild not premium")
)

var definition_addPremiumGuild = openai.FunctionDefinition{
	Name:        "addPremiumGuild",
	Description: "add a guild ID to the premium guild list.",

	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"id": {
				Type:        jsonschema.String,
				Description: "guild ID",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"id"},
	},
}

var definition_getPremiumGuilds = openai.FunctionDefinition{
	Name:        "getPremiumGuilds",
	Description: "get a list of all premium guilds",

	Parameters: jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
		Required:   []string{},
	},
}

var definition_removePremiumGuild = openai.FunctionDefinition{
	Name:        "removePremiumGuild",
	Description: "remove a guild ID from the premium guild list.",

	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"id": {
				Type:        jsonschema.String,
				Description: "guild ID",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"id"},
	},
}

func (adm *Admin) handler_AddPremiumGuild(args string) (string, error) {
	msgMap, err := argsToMap(args)
	if err != nil {
		return "", err
	}

	err = adm.action_AddPremiumGuild(msgMap["id"].(string))

	if errors.Is(err, errGuildAlreadyPremium) {
		return "guild already premium", nil
	}

	if err != nil {
		return "", err
	}

	return "added successfully.", nil
}

func (adm *Admin) handler_GetPremiumGuilds(_ string) (string, error) {
	guilds, err := adm.action_GetPremiumGuilds()
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(guilds)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (adm *Admin) handler_RemovePremiumGuild(args string) (string, error) {
	msgMap, err := argsToMap(args)
	if err != nil {
		return "", err
	}

	err = adm.action_RemovePremiumGuild(msgMap["id"].(string))

	if errors.Is(err, errGuildNotPremium) {
		return "guild not premium", nil
	}

	if err != nil {
		return "", err
	}

	return "added successfully.", nil
}

func (adm *Admin) action_AddPremiumGuild(id string) error {
	if adm.PremiumDB.IsPremium(id) {
		return errGuildAlreadyPremium
	}

	logrus.WithField("guild_id", id).Infoln("ADDING GUILD AS PREMIUM")
	return adm.PremiumDB.Add(id)
}

type premiumGuildsResponse struct {
	Ids []string `json:"premium_guild_ids"`
}

func (adm *Admin) action_GetPremiumGuilds() (premiumGuildsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return premiumGuildsResponse{
		Ids: adm.PremiumDB.GetPremium(ctx),
	}, nil
}
func (adm *Admin) action_RemovePremiumGuild(id string) error {
	if !adm.PremiumDB.IsPremium(id) {
		return errGuildNotPremium
	}

	logrus.WithField("guild_id", id).Infoln("REMOVING GUILD AS PREMIUM")
	return adm.PremiumDB.Delete(id)
}
