package depechebot

import (
	"log"
	"time"
	"encoding/json"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
	db "github.com/DepecheBot/depechebot/database"
	models "github.com/DepecheBot/depechebot/database/models"
)

const (
	ChatChanBufSize int = 1000
	TelegramTimeout = 60 //msec
)

type Chat models.Chat
type ChatChan struct {
	*models.Chat
	channel chan tgbotapi.Update
}

type ChatIDType int64

var chats = map[int]*ChatChan {}
var bot *tgbotapi.BotAPI
var SendChan chan tgbotapi.Chattable

func DepecheBot() {

}

func init() {
}

func Init(telegramToken string, dbName string,
	StatesConfigPrivate map[StateName]StateActions,
	StatesConfigGroup map[StateName]StateActions,
	commonLog func(tgbotapi.Update),
	chatLog func(tgbotapi.Update, Chat)) {

	db.InitDB(dbName)
	defer db.DB.Close()
	err := db.LoadChatsFromDB()
	check(err)

	for _, chat := range db.Chats {
		chats[chat.ChatID] = &ChatChan{chat, nil}
	}


	bot, err = tgbotapi.NewBotAPI(telegramToken)
	check(err)
	log.Printf("Authorized on account %s", bot.Self.UserName)
	
	SendChan = make(chan tgbotapi.Chattable, 100)
	go processSendChan()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = TelegramTimeout
	updates, err := bot.GetUpdatesChan(u)

	processUpdates(updates, StatesConfigPrivate, StatesConfigGroup, commonLog, chatLog)
}

func processUpdates(updates <-chan tgbotapi.Update,
	StatesConfigPrivate map[StateName]StateActions,
	StatesConfigGroup map[StateName]StateActions,
	commonLog func(tgbotapi.Update),
	chatLog func(tgbotapi.Update, Chat)) {

	for update := range updates {

		commonLog(update)

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

		if chat.channel == nil {
			chat.channel = make(chan tgbotapi.Update, ChatChanBufSize)
			if chat.Type == "private" {
				go processChat(chats[chatID].Chat, chats[chatID].channel, StatesConfigPrivate, chatLog)
			} else {
				go processChat(chats[chatID].Chat, chats[chatID].channel, StatesConfigGroup, chatLog)
			}
		}

		select {
		case chat.channel <- update:
		default:
			log.Printf("Channel buffer for chat %v is full!", chatID)
			log.Println(chat.channel)
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
func processChat(chat *models.Chat,
	channel <-chan tgbotapi.Update,
	StatesConfig map[StateName]StateActions,
	chatLog func(tgbotapi.Update, Chat)) {

	var update tgbotapi.Update
	var state State
	var groups Groups

	for {
		err := json.Unmarshal([]byte(chat.State), &state)
		check(err)
		groups.Parameters = jsonMap(chat.Groups)

		if _, ok := StatesConfig[state.Name]; !ok {
			log.Panicf("No such state: %v", state.Name)
		}

		while := StatesConfig[state.Name].While
		if while != nil {
			update = while(channel)
			updateChat(update, chat)
			chatLog(update, Chat(*chat))
		}

		after := StatesConfig[state.Name].After
		if after != nil {
			after(Chat(*chat), update, &state, &groups) // todo: fix int64
			chat.State = marshal(state)
			chat.Groups = string(groups.Parameters)

			if state.Parameters != "{}" {
				log.Printf("State after: %v with parameters: %v", state.Name, state.Parameters)
			} else {
				log.Printf("State after: %v", state.Name)
			}
		}

		if !state.skipBefore {
			before := StatesConfig[state.Name].Before // todo: fix int64
			if before != nil {
				before(Chat(*chat)) // todo: fix int64
			}
		}
		chat.Save(db.DB)
	}
}

// goroutine
func processSendChan() {
	const (
		commonDelay = time.Second / 30
	)

	for msg := range SendChan {
		_, err := bot.Send(msg)
		if err != nil {
			//log.Panicf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
			log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
		}

		time.Sleep(commonDelay)
	}
}
