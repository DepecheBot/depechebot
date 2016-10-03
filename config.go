package depechebot

import (
	"log"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

//type LanguageType string
type Request struct {
	text string //map[LanguageType]string
	unprescribed bool
}
type Responser interface {
	Response() func(Chat, tgbotapi.Update) State
}
type Responsers []Responser
type ReqToRes map[Request]Responser
type State string
type Text string

type StateActions struct {
	Before func(Chat)
	While func(<-chan tgbotapi.Update) tgbotapi.Update
	After func(Chat, tgbotapi.Update) State
}

func NewText(s string) Text {
	return Text(s)
}

func NewState(s string) State {
	return State(s)
}

func NewRequest(s string) Request {
	return Request{
		text : s,
		unprescribed : false,
	}
}

func NewUnprescribedRequest() Request {
	return Request{
		text : "",
		unprescribed : true,
	}
}


const (
	StartState = State("START")
	NillState = State("")
)


func StateBefore(text Text, keyboard interface{}) func(chat Chat) {
	return func(chat Chat) {
		msg := tgbotapi.NewMessage(int64(chat.ChatID), string(text))
		switch keyboard := keyboard.(type) {
		default:
			msg.ReplyMarkup = keyboard
		case [][]Request:
			msg.ReplyMarkup = Keyboard(keyboard)
		case []Request:
			msg.ReplyMarkup = Keyboard([][]Request{keyboard})
		case Request:
			if keyboard == NewUnprescribedRequest() {
				msg.ReplyMarkup = tgbotapi.ReplyKeyboardHide{HideKeyboard : true}
			} else {
				msg.ReplyMarkup = Keyboard([][]Request{{keyboard}})
			}
		}

		SendChan <- msg
	}
}

func StateWhile() func(<-chan tgbotapi.Update) tgbotapi.Update {
	return func(channel <-chan tgbotapi.Update) tgbotapi.Update {
		return <-channel
	}
}

func StateAfter(responsers ...Responser) func(Chat, tgbotapi.Update) State {
	return Responsers(responsers).Response()
}

func (responsers Responsers) Response() func(Chat, tgbotapi.Update) State {
	return func(chat Chat, update tgbotapi.Update) State {
		var s State
		for _, responser := range responsers {
			s = responser.Response()(chat, update)
		}
		return s
	}
}

func (text Text) Response() func(Chat, tgbotapi.Update) State {
	return func(chat Chat, update tgbotapi.Update) State {
		if text != "" {
			msg := tgbotapi.NewMessage(int64(chat.ChatID), string(text))
			SendChan <- msg
		}
		return NillState
	}
}

func (state State) Response() func(Chat, tgbotapi.Update) State {
	return func(chat Chat, update tgbotapi.Update) State {
		return state
	}
}

func (responses ReqToRes) Response() func(Chat, tgbotapi.Update) State {
	return func(chat Chat, update tgbotapi.Update) State {
		response, ok := responses[NewRequest(update.Message.Text)]
		if !ok {
			response, ok = responses[NewUnprescribedRequest()]
			if !ok {
				log.Panicf("no state %v in states %v\n", update.Message.Text, responses)
			}
		}

		return response.Response()(chat, update)
	}
}

func Keyboard(keyboard [][]Request) tgbotapi.ReplyKeyboardMarkup {
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
