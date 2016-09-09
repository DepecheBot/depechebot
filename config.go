package depechebot

import (
	//"log"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type StateIDType string
type State struct {
	Before func(ChatIDType)
	While func()
	After func(ChatIDType, tgbotapi.Update)
}

