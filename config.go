package depechebot

import (
	"encoding/json"
	"fmt"
	"log"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

//type LanguageType string
type Request struct {
	Text         string //map[LanguageType]string
	unprescribed bool
}
type ResponseFunc func(Bot, Chat, tgbotapi.Update, *State, *Params)
type Responser interface {
	Response(Bot, Chat, tgbotapi.Update, *State, *Params)
}
type Responsers []Responser
type ReqToRes map[Request]Responser

//type State string
type StateName string
type jsonMap string
type Params jsonMap
type State struct {
	Name       StateName `json:"name"`
	Params     jsonMap   `json:"params"`
	skipBefore bool
}
type Text struct {
	Text      string
	ParseMode string
}
type Photo struct {
	Caption string
	FileID  string
}
type StateActions struct {
	Before func(Bot, Chat)
	While  func(Bot, <-chan Signal) Signal
	After  func(Bot, Chat, tgbotapi.Update, *State, *Params)
}

func NewText(s string) Text {
	return Text{Text: s}
}
func NewTextWithMarkdown(s string) Text {
	return Text{Text: s, ParseMode: tgbotapi.ModeMarkdown}
}
func NewPhoto(fileID string) Photo {
	return Photo{FileID: fileID}
}

func NewPhotoWithCaption(fileID string, caption string) Photo {
	return Photo{
		FileID:  fileID,
		Caption: caption,
	}
}

func NewState(s string) State {
	return State{
		Name:   StateName(s),
		Params: "{}",
	}
}

func NewParams(key, value string) Params {
	params := Params(jsonMap("{}"))
	params.Set(key, value)
	return params
}

func NewRequest(s string) Request {
	return Request{
		Text:         s,
		unprescribed: false,
	}
}

func NewUnprescribedRequest() Request {
	return Request{
		Text:         "",
		unprescribed: true,
	}
}

func (state State) SkippedBefore() State {
	state.skipBefore = true
	return state
}

func (state State) WithParam(key, value string) State {
	newState := state
	(&newState.Params).Set(key, value)
	return newState
}

func (state State) String() string {
	if state.Params != "{}" {
		return fmt.Sprintf("%v with params: %v", state.Name, state.Params)
	} else {
		return fmt.Sprintf("%v", state.Name)
	}
}

func (params *Params) AddParams(newParams Params) {
	var m1 map[string]string

	json.Unmarshal([]byte(newParams), &m1)
	//json.Unmarshal([]byte(params), &m2)
	for key, value := range m1 {
		params.Set(key, value)
	}
}

func (jm jsonMap) Get(key string) string {
	var m map[string]string

	check(json.Unmarshal([]byte(jm), &m))
	return m[key]
}

func (p Params) Get(key string) string {
	return jsonMap(p).Get(key)
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

func (p *Params) Set(key, value string) {
	(*jsonMap)(p).Set(key, value)
}

func (jm jsonMap) With(key, value string) jsonMap {
	newJM := jm
	(&newJM).Set(key, value)
	return newJM
}

func (p Params) With(key, value string) Params {
	return Params(jsonMap(p).With(key, value))
}

var (
	StartState = NewState("START")
)

func UniversalResponse(chat Chat, update tgbotapi.Update, state *State, params *Params) {
	//*state = StartState
	// todo: fixme!!! Need to initialize UniversalResponse in config
	*state = NewState("MAIN")
}

func StateBefore(text Text, keyboard interface{}) func(bot Bot, chat Chat) {
	return func(bot Bot, chat Chat) {
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
				msg.ReplyMarkup = tgbotapi.ReplyKeyboardHide{HideKeyboard: true}
			} else {
				msg.ReplyMarkup = Keyboard([][]Request{{keyboard}})
			}
		}

		bot.SendChan <- ChatSignal{msg, ChatIDType(chat.ChatID)}
	}
}

func StateWhile() func(Bot, <-chan Signal) Signal {
	return func(bot Bot, signalChan <-chan Signal) Signal {
		return <-signalChan
	}
}

func StateAfter(responsers ...Responser) func(Bot, Chat, tgbotapi.Update, *State, *Params) {
	return Responsers(responsers).Response
}

func (responsers Responsers) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	for _, responser := range responsers {
		responser.Response(bot, chat, update, state, params)
	}
}

func (text Text) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	if text.Text != "" {
		msg := tgbotapi.NewMessage(int64(chat.ChatID), text.Text)
		msg.ParseMode = text.ParseMode
		bot.SendChan <- ChatSignal{msg, ChatIDType(chat.ChatID)}
	}
}

func (photo Photo) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	msg := tgbotapi.NewPhotoShare(int64(chat.ChatID), photo.FileID)
	if photo.Caption != "" {
		msg.Caption = photo.Caption
	}
	bot.SendChan <- ChatSignal{msg, ChatIDType(chat.ChatID)}
}

func (newState State) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	*state = newState
}

func (newParams Params) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	params.AddParams(newParams)
}

func (responses ReqToRes) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	response, ok := responses[NewRequest(update.Message.Text)]
	if !ok {
		response, ok = responses[NewUnprescribedRequest()]
		if !ok {
			//response = UniversalResponse
			UniversalResponse(chat, update, state, params)
			log.Printf("no response %v in responses %v\n", update.Message.Text, responses)
			return
		}
	}

	response.Response(bot, chat, update, state, params)
}

func (responseFunc ResponseFunc) Response(bot Bot, chat Chat, update tgbotapi.Update, state *State, params *Params) {
	responseFunc(bot, chat, update, state, params)
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
