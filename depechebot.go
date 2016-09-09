package depechebot

import (
	"log"
	"time"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
	db "github.com/DepecheBot/depechebot/database"
	models "github.com/DepecheBot/depechebot/database/models"
)

const (
	ChatChanBufSize int = 100
	TelegramTimeout = 60 //msec
)

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

func Init(telegramToken string, dbName string, StatesConfig map[StateIDType]State) {
	var err error

	db.InitDB(dbName)
	defer db.DB.Close()
	err = db.LoadChatsFromDB()
	if err != nil {
		log.Panic(err)
	}

	for _, chat := range db.Chats {
		chats[chat.ChatID] = &ChatChan{chat, nil}
	}


	bot, err = tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)
	
	SendChan = make(chan tgbotapi.Chattable, 100)
	go func() {
		for msg := range SendChan {
			bot.Send(msg)
		}
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = TelegramTimeout
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {

		adminLog(update)
		
		// todo: update.Query and so on...
		if update.Message == nil {
			log.Println(update)
			continue
		}

		chatID := int(update.Message.Chat.ID) // todo: fix int() for 32-bit
		chat, ok := chats[chatID]
		if ok {
			chat.UserName = update.Message.From.UserName
			chat.FirstName = update.Message.From.FirstName
			chat.LastName = update.Message.From.LastName
			chat.LastTime = time.Now().String()
		} else {
			chat = &ChatChan{}
			chats[chatID] = chat
			chat.Chat = &models.Chat{
				ChatID: chatID,
				UserID: update.Message.From.ID,
				UserName: update.Message.From.UserName,
				FirstName: update.Message.From.FirstName,
				LastName: update.Message.From.LastName,
				OpenTime: time.Now().String(),
				LastTime: time.Now().String(),
				Groups: "",
				State: "START",
			}
		}
		
		if chat.channel == nil {
			chat.channel = make(chan tgbotapi.Update, ChatChanBufSize)
			go processChat(chats[chatID].Chat, chats[chatID].channel, bot, StatesConfig)
		}


		select {
		case chat.channel <- update:
		default:
			log.Printf("Channel buffer for chat %v is full!", chatID)
			log.Println(chat.channel)
		}
	}
}

func adminLog(update tgbotapi.Update) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
}

func processChat(chat *models.Chat, channel <-chan tgbotapi.Update,
	bot *tgbotapi.BotAPI, StatesConfig map[StateIDType]State) {
	for {
		update := <-channel
		
		StatesConfig[StateIDType(chat.State)].After(ChatIDType(chat.ChatID), update) // todo: fix int64

		// todo: switch state here

		StatesConfig[StateIDType(chat.State)].Before(ChatIDType(chat.ChatID)) // todo: fix int64

		chat.Save(db.DB)

	}

}

