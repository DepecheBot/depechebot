package depechebot

import (
	"log"
	"encoding/json"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

//type LanguageType string
type Request struct {
	Text string //map[LanguageType]string
	unprescribed bool
}
type Responser interface {
	Response() func(Chat, tgbotapi.Update, *State)
}
type Responsers []Responser
type ReqToRes map[Request]Responser
//type State string
type StateName string
type State struct {
	Name StateName `json:"name"`
	Parameters string `json:"parameters"`
	skipBefore bool `json:"-"`
}
type Text string

type StateActions struct {
	Before func(Chat)
	While func(<-chan tgbotapi.Update) tgbotapi.Update
	After func(Chat, tgbotapi.Update, *State)
}

func NewText(s string) Text {
	return Text(s)
}

func NewState(s string) State {
	return State{
		Name : StateName(s),
		Parameters : "{}",
	}
}

func NewRequest(s string) Request {
	return Request{
		Text : s,
		unprescribed : false,
	}
}

func NewUnprescribedRequest() Request {
	return Request{
		Text : "",
		unprescribed : true,
	}
}

func (state State) SkippedBefore() State {
	state.skipBefore = true
	return state
}

func (state State) GetParameter(param string) string {
	var parameters map[string]string

	check(json.Unmarshal([]byte(state.Parameters), &parameters))
	return parameters[param]
}

func (state State) WithParameter(param string, value string) State {
	newState := state
	var parameters map[string]string

	err := json.Unmarshal([]byte(newState.Parameters), &parameters)
	if err != nil {
		log.Println("parameters: %v", newState.Parameters)
		log.Panic(err)
	}
	parameters[param] = value
	newState.Parameters = marshal(parameters)

	return newState
}

var (
	StartState = NewState("START")
)


func UniversalResponse(chat Chat, update tgbotapi.Update, state *State) {
	*state = StartState
}

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

func StateAfter(responsers ...Responser) func(Chat, tgbotapi.Update, *State) {
	return Responsers(responsers).Response()
}

func (responsers Responsers) Response() func(Chat, tgbotapi.Update, *State) {
	return func(chat Chat, update tgbotapi.Update, state *State) {
		for _, responser := range responsers {
			responser.Response()(chat, update, state)
		}
	}
}

func (text Text) Response() func(Chat, tgbotapi.Update, *State) {
	return func(chat Chat, update tgbotapi.Update, state *State) {
		if text != "" {
			msg := tgbotapi.NewMessage(int64(chat.ChatID), string(text))
			SendChan <- msg
		}
	}
}

func (newState State) Response() func(Chat, tgbotapi.Update, *State) {
	return func(chat Chat, update tgbotapi.Update, state *State) {
		*state = newState
	}
}

func (responses ReqToRes) Response() func(Chat, tgbotapi.Update, *State) {
	return func(chat Chat, update tgbotapi.Update, state *State) {
		response, ok := responses[NewRequest(update.Message.Text)]
		if !ok {
			response, ok = responses[NewUnprescribedRequest()]
			if !ok {
				//response = UniversalResponse
				UniversalResponse(chat, update, state)
				return
				log.Printf("no response %v in responses %v\n", update.Message.Text, responses)
			}
		}

		response.Response()(chat, update, state)
	}
}

func Keyboard(keyboard [][]Request) tgbotapi.ReplyKeyboardMarkup {
	var Keyboard [][]tgbotapi.KeyboardButton
	for _, row := range keyboard {
		var Row []tgbotapi.KeyboardButton
		for _, button := range row {
			Row = append(Row, tgbotapi.NewKeyboardButton(button.Text))
		}
		Keyboard = append(Keyboard, Row)
	}
	return tgbotapi.NewReplyKeyboard(Keyboard...)
}
