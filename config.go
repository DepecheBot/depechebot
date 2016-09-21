package depechebot

import (
	"log"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type StateIDType string
type StateActions struct {
	Before func(Chat)
	While func()
	After func(Chat, tgbotapi.Update) StateIDType
}

func StateBefore(message string, keyboard interface{}) func(chat Chat) {
	return func(chat Chat) {
		msg := tgbotapi.NewMessage(int64(chat.ChatID), message)
		switch keyboard := keyboard.(type) {
		default:
			msg.ReplyMarkup = keyboard
		case [][]string:
			msg.ReplyMarkup = Keyboard(keyboard)
		case []string:
			msg.ReplyMarkup = Keyboard([][]string{keyboard})
		}

		SendChan <- msg
	}
}

func StateAfter(msg string, states interface{}) func(Chat, tgbotapi.Update) StateIDType {
	switch states := states.(type) {
	default:
		log.Panicf("unexpected type %T\n", states)
		return nil
	case string:
		return func(chat Chat, update tgbotapi.Update) StateIDType {
			if msg != "" {
				msg := tgbotapi.NewMessage(int64(chat.ChatID), msg)
				SendChan <- msg
			}

			return StateIDType(states)
		}
	case map[string]string:
		return func(chat Chat, update tgbotapi.Update) StateIDType {
			state, ok := states[update.Message.Text]
			if !ok {
				state, ok = states["default"]
				if !ok {
					log.Panicf("no state %v in states %v\n", update.Message.Text, states)
				}
			}

			if msg != "" {
				msg := tgbotapi.NewMessage(int64(chat.ChatID), msg)
				SendChan <- msg
			}

			return StateIDType(state)
		}
	}
}

func Keyboard(keyboard [][]string) tgbotapi.ReplyKeyboardMarkup {
	var Keyboard [][]tgbotapi.KeyboardButton
	for _, row := range keyboard {
		var Row []tgbotapi.KeyboardButton
		for _, button := range row {
			Row = append(Row, tgbotapi.NewKeyboardButton(button))
		}
		Keyboard = append(Keyboard, Row)
	}
	return tgbotapi.NewReplyKeyboard(Keyboard...)
}
