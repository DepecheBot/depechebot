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
type ResponseFunc func() func(Chat, tgbotapi.Update, *State)
type Responsers []Responser
type ReqToRes map[Request]Responser
//type State string
type StateName string
type jsonMap string
type State struct {
	Name StateName `json:"name"`
	Parameters jsonMap `json:"parameters"`
	skipBefore bool `json:"-"`
}

type Text struct {
	Text string
	ParseMode string
}

type Photo struct {
	Caption string
	FileID string
}

type StateActions struct {
	Before func(Chat)
	While func(<-chan tgbotapi.Update) tgbotapi.Update
	After func(Chat, tgbotapi.Update, *State)
}

func NewText(s string) Text {
	return Text{Text : s}
}
func NewTextWithMarkdown(s string) Text {
	return Text{Text : s, ParseMode : "Markdown"}
}
func NewPhoto(fileID string) Photo {
	return Photo{FileID : fileID}
}

func NewPhotoWithCaption(fileID string, caption string) Photo {
	return Photo{
		FileID : fileID,
		Caption : caption,
	}
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

func (state State) WithParameter(key, value string) State {
	newState := state
	(&newState.Parameters).Set(key, value)
	return newState
}

func (jm jsonMap) Get(key string) string {
	var m map[string]string

	check(json.Unmarshal([]byte(jm), &m))
	return m[key]
}

func (jm *jsonMap) Set(key, value string) {
	var m map[string]string

	err := json.Unmarshal([]byte(*jm), &m)
	if err != nil {
		log.Panicf("jm: %v, err: %v\n", jm, err)
	}
	m[key] = value
	*jm = jsonMap(marshal(m))
}

func (jm jsonMap) With(key, value string) jsonMap {
	newJM := jm
	(&newJM).Set(key, value)
	return newJM
}

var (
	StartState = NewState("START")
)


func UniversalResponse(chat Chat, update tgbotapi.Update, state *State) {
	//*state = StartState
	// todo: fixme!!! Need to initialize UniversalResponse in config
	*state = NewState("MAIN")
}

func StateBefore(text Text, keyboard interface{}) func(chat Chat) {
	return func(chat Chat) {
		msg := tgbotapi.NewMessage(int64(chat.ChatID), text.Text)
		msg.ParseMode = text.ParseMode
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
		if text.Text != "" {
			msg := tgbotapi.NewMessage(int64(chat.ChatID), text.Text)
			msg.ParseMode = text.ParseMode
			SendChan <- msg
		}
	}
}

func (photo Photo) Response() func(Chat, tgbotapi.Update, *State) {
	return func(chat Chat, update tgbotapi.Update, state *State) {
		msg := tgbotapi.NewPhotoShare(int64(chat.ChatID), photo.FileID)
		if photo.Caption != "" {
			msg.Caption = photo.Caption
		}
		SendChan <- msg
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

func (responseFunc ResponseFunc) Response() func(Chat, tgbotapi.Update, *State) {
	return responseFunc()
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
