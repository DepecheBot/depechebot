package depechebot

import (
	"log"
	"time"
	"encoding/json"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
	db "github.com/depechebot/depechebot/database"
	models "github.com/depechebot/depechebot/database/models"
)

const (
	chatChanBufSize = 100
	sendChanBufSize = 1000
	sendBroadChanBufSize = 100
	telegramTimeout = 60 //msec
)

type Chat models.Chat
// todo: split, do we need to store it together?
type ChatChan struct {
	*models.Chat
	signalChan chan Signal
}

type ChatIDType int64

var chats = map[int]*ChatChan {}
var bot *tgbotapi.BotAPI

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
var SendChan chan ChatSignal
var SendBroadChan chan BroadSignal

var StatesConfigPrivate map[StateName]StateActions
var StatesConfigGroup map[StateName]StateActions
var CommonLog func(tgbotapi.Update)
var ChatLog func(tgbotapi.Update, Chat)


func DepecheBot() {

}

func init() {
}

func Init(telegramToken string, dbName string) {

	db.InitDB(dbName)
	defer db.DB.Close()
	err := db.LoadChatsFromDB()
	check(err)

	var i int
	var chat *models.Chat
	for i, chat = range db.Chats {
		chats[chat.ChatID] = &ChatChan{chat, make(chan Signal, chatChanBufSize)}
		go processChat(chats[chat.ChatID].Chat, chats[chat.ChatID].signalChan)
	}
	log.Printf("Loaded %v chats from DB file %v\n", i + 1, dbName)

	bot, err = tgbotapi.NewBotAPI(telegramToken)
	check(err)
	log.Printf("Authorized on account %s", bot.Self.UserName)
	
	SendChan = make(chan ChatSignal, sendChanBufSize)
	go processSendChan()

	SendBroadChan = make(chan BroadSignal, sendBroadChanBufSize)
	go processSendBroadChan()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = telegramTimeout
	updates, err := bot.GetUpdatesChan(u)

	processUpdates(updates)
}

func processUpdates(updates <-chan tgbotapi.Update) {

	for update := range updates {

		CommonLog(update)

		// todo: update.Query and so on...
		if update.Message == nil {
			continue
		}

		chatID := int(update.Message.Chat.ID) // todo: fix int() for 32-bit
		chat, ok := chats[chatID]

		if !ok {
			chat = &ChatChan{}
			chats[chatID] = chat

			chat.Chat = &models.Chat{
				ChatID: chatID,
				Abandoned: bool2int(false),
				Type : update.Message.Chat.Type,
				UserID: update.Message.From.ID,
				UserName: update.Message.From.UserName,
				FirstName: update.Message.From.FirstName,
				LastName: update.Message.From.LastName,
				OpenTime: time.Now().String(),
				LastTime: time.Now().String(),
				Groups: "{}",
				State: marshal(StartState),
			}
		}

		if chat.signalChan == nil {
			chat.signalChan = make(chan Signal, chatChanBufSize)
			go processChat(chat.Chat, chat.signalChan)
		}

		select {
		case chat.signalChan <- update:
		default:
			log.Printf("Channel buffer for chat %v is full!", chatID)
			log.Println(chat.signalChan) // todo: print buffer here, not interface{}
		}
	}
}


func updateChat(update tgbotapi.Update, chat *models.Chat) {

	var abandoned = false
	// checked either bot is kicked itself or he is alone now
	if update.Message.LeftChatMember != nil {
		if update.Message.LeftChatMember.ID == bot.Self.ID {
			abandoned = true
		} else {
			count, err := bot.GetChatMembersCount(update.Message.Chat.ChatConfig())
			check(err)
			if count == 1 {
				abandoned = true
				bot.LeaveChat(update.Message.Chat.ChatConfig())
			}
		}
	}
	if update.Message.NewChatMember != nil &&
		update.Message.NewChatMember.ID == bot.Self.ID {
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
func processChat(chat *models.Chat, signalChan <-chan Signal) {
	var update tgbotapi.Update
	var state State
	var groups Groups
	var statesConfig map[StateName]StateActions

	if chat.Type == "private" {
		statesConfig = StatesConfigPrivate
	} else {
		statesConfig = StatesConfigGroup
	}

	for {
		err := json.Unmarshal([]byte(chat.State), &state)
		check(err)
		groups.Parameters = jsonMap(chat.Groups)

		if _, ok := statesConfig[state.Name]; !ok {
			log.Panicf("No such state: %v", state.Name)
		}

		while := statesConfig[state.Name].While
		after := statesConfig[state.Name].After
		if while != nil {
		WhileLoop:
			for {
				signal := while(signalChan)

				switch signal := signal.(type) {
				case tgbotapi.Update:
					update = signal
					updateChat(update, chat)
					ChatLog(update, Chat(*chat))
					break WhileLoop
				case State:
					state = signal
					chat.State = marshal(state)
					log.Printf("    Interrupted with state: %v", state)
					goto BeforeLabel
				case tgbotapi.MessageConfig:
					msg := signal
					msg.ChatID = int64(chat.ChatID)
					_, err := bot.Send(msg)
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
					_, err := bot.Send(msg)
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
			after(Chat(*chat), update, &state, &groups) // todo: fix int64
			chat.State = marshal(state)
			chat.Groups = string(groups.Parameters)

			log.Printf("    State after: %v", state)
		}

	BeforeLabel:
		if !state.skipBefore {
			before := statesConfig[state.Name].Before // todo: fix int64
			if before != nil {
				before(Chat(*chat)) // todo: fix int64
			}
		}

		// defer() this?
		chat.Save(db.DB)
	}
}

// goroutine
func processSendChan() {
	const (
		commonDelay = time.Second / 30
	)

	for chatSignal := range SendChan {
		// todo: fix race!
		chats[int(chatSignal.ChatID)].signalChan <- chatSignal.Signal
		time.Sleep(commonDelay)
	}
}

// goroutine
func processSendBroadChan() {
	const (
		commonDelay = time.Second / 30
	)

	for broadSignal := range SendBroadChan {
		for _, chatID := range broadSignal.List {
			// todo: fix race!
			chats[int(chatID)].signalChan <- broadSignal.Signal
			time.Sleep(commonDelay)
		}
	}
}
