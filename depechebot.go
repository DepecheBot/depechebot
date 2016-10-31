package depechebot

import (
	"encoding/json"
	"log"
	"time"

	db "github.com/depechebot/depechebot/database"
	models "github.com/depechebot/depechebot/database/models"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const (
	chatChanBufSize      = 100
	sendChanBufSize      = 1000
	sendBroadChanBufSize = 100
	telegramTimeout      = 60 //msec
)

type Chat models.Chat

// todo: split, do we need to store it together?
type ChatChan struct {
	*models.Chat
	signalChan chan Signal
}

type ChatIDType int64

// Signal could be either tgbotapi.Chattable (or tgbotapi.MessageConfig),
// State (interrupt state) or tgbotapi.Update
type Signal interface{}
type ChatSignal struct {
	Signal
	ChatID ChatIDType
}
type BroadSignal struct {
	Signal
	List []ChatIDType
}

type Config struct {
	TelegramToken string
	//AdminLog func()
	CommonLog           func(tgbotapi.Update)
	ChatLog             func(tgbotapi.Update, Chat)
	StatesConfigPrivate map[StateName]StateActions
	StatesConfigGroup   map[StateName]StateActions
	DBName              string
}

type Bot struct {
	SendChan      chan ChatSignal
	SendBroadChan chan BroadSignal

	config Config
	chats  map[int]*ChatChan
	api    *tgbotapi.BotAPI
}

func New(c Config) (Bot, error) {
	var err error

	bot := Bot{config: c}
	bot.chats = make(map[int]*ChatChan)
	bot.SendChan = make(chan ChatSignal, sendChanBufSize)
	bot.SendBroadChan = make(chan BroadSignal, sendBroadChanBufSize)

	bot.api, err = tgbotapi.NewBotAPI(bot.config.TelegramToken)
	if err != nil {
		return bot, err
	}

	return bot, nil
}

func (b Bot) Run() {
	var err error

	log.Printf("Authorized on account %s", b.api.Self.UserName)

	db.InitDB(b.config.DBName)
	defer db.DB.Close()
	err = db.LoadChatsFromDB()
	check(err)

	var i int
	var chat *models.Chat
	for i, chat = range db.Chats {
		b.chats[chat.ChatID] = &ChatChan{chat, make(chan Signal, chatChanBufSize)}
	}
	log.Printf("Loaded %v chats from DB file %v\n", i+1, b.config.DBName)

	for _, chat = range db.Chats {
		go b.processChat(b.chats[chat.ChatID].Chat, b.chats[chat.ChatID].signalChan)
	}

	go b.processSendChan()
	go b.processSendBroadChan()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = telegramTimeout
	updates, err := b.api.GetUpdatesChan(u)

	b.processUpdates(updates)
}

func (b Bot) processUpdates(updates <-chan tgbotapi.Update) {

	for update := range updates {

		b.config.CommonLog(update)

		// todo: update.Query and so on...
		if update.Message == nil {
			continue
		}

		chatID := int(update.Message.Chat.ID) // todo: fix int() for 32-bit
		chat, ok := b.chats[chatID]

		if !ok {
			chat = &ChatChan{}
			b.chats[chatID] = chat

			chat.Chat = &models.Chat{
				ChatID:    chatID,
				Abandoned: bool2int(false),
				Type:      update.Message.Chat.Type,
				UserID:    update.Message.From.ID,
				UserName:  update.Message.From.UserName,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
				OpenTime:  time.Now().String(),
				LastTime:  time.Now().String(),
				State:     marshal(StartState),
				Params:    "{}",
			}
		}

		if chat.signalChan == nil {
			chat.signalChan = make(chan Signal, chatChanBufSize)
			go b.processChat(chat.Chat, chat.signalChan)
		}

		select {
		case chat.signalChan <- update:
		default:
			log.Printf("Channel buffer for chat %v is full!", chatID)
			log.Println(chat.signalChan) // todo: print buffer here, not interface{}
		}
	}
}

func (b Bot) updateChat(update tgbotapi.Update, chat *models.Chat) {

	var abandoned = false
	// checked either bot is kicked itself or he is alone now
	if update.Message.LeftChatMember != nil {
		if update.Message.LeftChatMember.ID == b.api.Self.ID {
			abandoned = true
		} else {
			count, err := b.api.GetChatMembersCount(update.Message.Chat.ChatConfig())
			check(err)
			if count == 1 {
				abandoned = true
				b.api.LeaveChat(update.Message.Chat.ChatConfig())
			}
		}
	}
	if update.Message.NewChatMember != nil &&
		update.Message.NewChatMember.ID == b.api.Self.ID {
		abandoned = false
	}
	if update.Message.MigrateToChatID != 0 {
		// todo: need to do more here to migrate
		abandoned = true
	}
	if update.Message.MigrateFromChatID != 0 {
		// todo: need to do more here to migrate
	}

	chat.Abandoned = bool2int(abandoned)
	chat.UserName = update.Message.From.UserName
	chat.FirstName = update.Message.From.FirstName
	chat.LastName = update.Message.From.LastName
	chat.LastTime = time.Now().String()

	// todo: is it correct?
	if abandoned {
		chat.State = marshal(StartState)
	}
}

// goroutine
func (b Bot) processChat(chat *models.Chat, signalChan <-chan Signal) {
	var update tgbotapi.Update
	var state State
	var params Params
	var statesConfig map[StateName]StateActions

	if chat.Type == "private" {
		statesConfig = b.config.StatesConfigPrivate
	} else {
		statesConfig = b.config.StatesConfigGroup
	}

	for {
		err := json.Unmarshal([]byte(chat.State), &state)
		check(err)
		params = Params(jsonMap(chat.Params))

		if _, ok := statesConfig[state.Name]; !ok {
			log.Panicf("No such state: %v", state.Name)
		}

		while := statesConfig[state.Name].While
		after := statesConfig[state.Name].After
		if while != nil {
		WhileLoop:
			for {
				signal := while(b, signalChan)

				switch signal := signal.(type) {
				case tgbotapi.Update:
					update = signal
					b.updateChat(update, chat)
					b.config.ChatLog(update, Chat(*chat))
					break WhileLoop
				case State:
					state = signal
					chat.State = marshal(state)
					log.Printf("    Interrupted with state: %v", state)
					goto BeforeLabel
				case tgbotapi.MessageConfig:
					msg := signal
					msg.ChatID = int64(chat.ChatID)
					_, err := b.api.Send(msg)
					if err != nil {
						log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
						if err.Error() == "forbidden" {
							chat.Abandoned = bool2int(true)
						}
					}
					continue WhileLoop
				case tgbotapi.Chattable: // todo: leave only one of MessageConfig and Chattable
					msg := signal
					// fix ChatID in this message!!
					_, err := b.api.Send(msg)
					if err != nil {
						log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
						if err.Error() == "forbidden" {
							chat.Abandoned = bool2int(true)
						}
					}
					continue WhileLoop
				default:
					log.Panicf("Should be either Update, State, MessageConfig or Chattable")
				}
			}
		}

		// todo: consider chat.Abandoned from here?
		if after != nil {
			after(b, Chat(*chat), update, &state, &params) // todo: fix int64
			chat.State = marshal(state)
			chat.Params = string(params)

			log.Printf("    State after: %v", state)
		}

	BeforeLabel:
		if !state.skipBefore {
			before := statesConfig[state.Name].Before // todo: fix int64
			if before != nil {
				before(b, Chat(*chat)) // todo: fix int64
			}
		}

		// defer() this?
		chat.Save(db.DB)
	}
}

// goroutine
func (b Bot) processSendChan() {
	const (
		commonDelay = time.Second / 30
	)

	for chatSignal := range b.SendChan {
		// todo: fix race!
		b.chats[int(chatSignal.ChatID)].signalChan <- chatSignal.Signal
		time.Sleep(commonDelay)
	}
}

// goroutine
func (b Bot) processSendBroadChan() {
	const (
		commonDelay = time.Second / 30
	)

	for broadSignal := range b.SendBroadChan {
		for _, chatID := range broadSignal.List {
			// todo: fix race!
			b.chats[int(chatID)].signalChan <- broadSignal.Signal
			time.Sleep(commonDelay)
		}
	}
}
