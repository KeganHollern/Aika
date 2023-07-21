package aika

import (
	"aika/actions"
	"aika/premium"
)

// controls if a DM is considered a premium chat
const DMsArePremium = true

type ChatContext struct {
	PremiumDB *premium.Servers
	CID       string // channel id
	GID       string // guild id
}

func (ctx *ChatContext) isPremiumChat() bool {
	if ctx.GID == "" { // only blank in DMs
		return DMsArePremium
	}
	return ctx.PremiumDB.IsPremium(ctx.GID)
}
func (ctx *ChatContext) isAdminChat() bool {
	return ctx.CID == "1127006446723276860" ||
		(ctx.CID == "1131772935414235197" && ctx.GID == "1092965539346907156")
}
func (ctx *ChatContext) getLanguageModel() LanguageModel {
	if ctx.isPremiumChat() || ctx.isAdminChat() {
		return LanguageModel_GPT4
	}

	// TODO: check if GID is in the PREMIUM list if YES then 4
	return LanguageModel_GPT35
}

func (ctx *ChatContext) getExtraChatFunctions() []actions.Function {
	extraFunctions := []actions.Function{}

	// admin commands
	if ctx.isAdminChat() {
		adm := &actions.Admin{PremiumDB: ctx.PremiumDB}
		extraFunctions = append(extraFunctions, adm.GetFunction_AddPremiumGuild())
		extraFunctions = append(extraFunctions, adm.GetFunction_GetPremiumGuilds())
		extraFunctions = append(extraFunctions, adm.GetFunction_RemovePremiumGuild())
	}

	return extraFunctions
}
