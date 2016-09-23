package depechebot

import (
	"log"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

//type LanguageType string
type Answer struct {
	text string //map[LanguageType]string
	unprescribed bool
}


func NewAnswer(s string) Answer {
	return Answer{
		text : s,
		unprescribed : false,
	}
}

func NewUnprescribedAnswer() Answer {
	return Answer{
		text : "",
		unprescribed : true,
	}
}

type StateID string
const (
	StartState = StateID(string(iota))
)

type StateActions struct {
	Before func(Chat)
	While func(<-chan tgbotapi.Update) tgbotapi.Update
	After func(Chat, tgbotapi.Update) StateID
}

func StateBefore(message string, keyboard interface{}) func(chat Chat) {
	return func(chat Chat) {
		msg := tgbotapi.NewMessage(int64(chat.ChatID), message)
		switch keyboard := keyboard.(type) {
		default:
			msg.ReplyMarkup = keyboard
		case [][]Answer:
			msg.ReplyMarkup = Keyboard(keyboard)
		case []Answer:
			msg.ReplyMarkup = Keyboard([][]Answer{keyboard})
		}

		SendChan <- msg
	}
}

func StateWhile() func(<-chan tgbotapi.Update) tgbotapi.Update {
	return func(channel <-chan tgbotapi.Update) tgbotapi.Update {
		return <-channel
	}
}

func StateAfter(msg string, states interface{}) func(Chat, tgbotapi.Update) StateID {
	switch states := states.(type) {
	default:
		log.Panicf("unexpected type %T\n", states)
		return nil
	case StateID:
		return func(chat Chat, update tgbotapi.Update) StateID {
			if msg != "" {
				msg := tgbotapi.NewMessage(int64(chat.ChatID), msg)
				SendChan <- msg
			}

			return StateID(states)
		}
	case map[Answer]StateID:
		return func(chat Chat, update tgbotapi.Update) StateID {
			state, ok := states[NewAnswer(update.Message.Text)]
			if !ok {
				state, ok = states[NewUnprescribedAnswer()]
				if !ok {
					log.Panicf("no state %v in states %v\n", update.Message.Text, states)
				}
			}

			if msg != "" {
				msg := tgbotapi.NewMessage(int64(chat.ChatID), msg)
				SendChan <- msg
			}

			return StateID(state)
		}
	}
}

func Keyboard(keyboard [][]Answer) tgbotapi.ReplyKeyboardMarkup {
	var Keyboard [][]tgbotapi.KeyboardButton
	for _, row := range keyboard {
		var Row []tgbotapi.KeyboardButton
		for _, button := range row {
			Row = append(Row, tgbotapi.NewKeyboardButton(button.text))
		}
		Keyboard = append(Keyboard, Row)
	}
	return tgbotapi.NewReplyKeyboard(Keyboard...)
}
