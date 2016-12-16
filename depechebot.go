package depechebot

import (
	"log"
	"sync"
	"time"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const (
	chatChanBufSize      = 100
	sendChanBufSize      = 1000
	sendBroadChanBufSize = 100
	telegramTimeout      = 60 //msec
)

// Signal could be either tgbotapi.Chattable (or tgbotapi.MessageConfig/PhotoConfig),
// State (interrupt state) or tgbotapi.Update
type Signal interface{}
type ChatSignal struct {
	Signal
	ChatID ChatID
}
type BroadSignal struct {
	Signal
	List []ChatID
}

type Config struct {
	TelegramToken string
	//AdminLog func()
	CommonLog           func(tgbotapi.Update)
	ChatLog             func(Bot, tgbotapi.Update, Chat)
	StatesConfigPrivate map[StateName]StateActions
	StatesConfigGroup   map[StateName]StateActions
	Model               Model
}

type Bot struct {
	SendChan      chan ChatSignal
	SendBroadChan chan BroadSignal
	Config

	chatsChans struct {
		*sync.RWMutex
		m map[ChatID]chan Signal
	}
	api *tgbotapi.BotAPI
}

func New(c Config) (Bot, error) {
	var err error

	bot := Bot{Config: c}
	bot.chatsChans.RWMutex = &sync.RWMutex{}
	bot.chatsChans.m = make(map[ChatID]chan Signal)
	bot.SendChan = make(chan ChatSignal, sendChanBufSize)
	bot.SendBroadChan = make(chan BroadSignal, sendBroadChanBufSize)

	bot.api, err = tgbotapi.NewBotAPI(bot.Config.TelegramToken)
	if err != nil {
		return bot, err
	}

	return bot, nil
}

func (b Bot) Run() {
	var err error

	log.Printf("Authorized on account %s", b.api.Self.UserName)

	chatIDs, err := b.Config.Model.Init()
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Loaded %v chats\n", len(chatIDs))

	for _, chatID := range chatIDs {
		b.chatsChans.m[chatID] = make(chan Signal, chatChanBufSize)
	}

	for _, chatID := range chatIDs {
		b.chatsChans.RLock()
		go b.processChat(chatID, b.chatsChans.m[chatID])
		b.chatsChans.RUnlock()
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

		b.Config.CommonLog(update)

		// todo: update.Query and so on...
		if update.Message == nil {
			continue
		}

		chatID := ChatID(update.Message.Chat.ID)
		b.chatsChans.RLock()
		chatChan, ok := b.chatsChans.m[chatID]
		b.chatsChans.RUnlock()

		if !ok {
			chat := &Chat{
				ChatID:    chatID,
				Abandoned: false,
				Type:      update.Message.Chat.Type,
				UserID:    update.Message.From.ID,
				UserName:  update.Message.From.UserName,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
				OpenTime:  time.Now(),
				LastTime:  time.Now(),
				State:     StartState,
				Params:    Params{},
			}
			err := b.Config.Model.Insert(chat)
			if err != nil {
				log.Panic(err)
			}
			chatChan = make(chan Signal, chatChanBufSize)
			b.chatsChans.Lock()
			b.chatsChans.m[chatID] = chatChan
			b.chatsChans.Unlock()
			go b.processChat(chatID, chatChan)
		}

		select {
		case chatChan <- update:
		default:
			log.Printf("Channel buffer for chat %v is full!", chatID)
			log.Println(chatChan) // todo: print buffer here, not interface{}
		}
	}
}

func (b Bot) updateChat(update tgbotapi.Update, chat *Chat) {

	var abandoned = false
	// checked either bot is kicked itself or he is alone now
	if update.Message.LeftChatMember != nil {
		if update.Message.LeftChatMember.ID == b.api.Self.ID {
			abandoned = true
		} else {
			count, err := b.api.GetChatMembersCount(update.Message.Chat.ChatConfig())
			if err != nil {
				log.Panic(err)
			}
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

	chat.Abandoned = abandoned
	chat.UserName = update.Message.From.UserName
	chat.FirstName = update.Message.From.FirstName
	chat.LastName = update.Message.From.LastName
	chat.LastTime = time.Now()

	// todo: is it correct?
	if abandoned {
		chat.State = StartState
	}
}

// goroutine
func (b Bot) processChat(chatID ChatID, signalChan <-chan Signal) {
	var update tgbotapi.Update
	var statesConfig map[StateName]StateActions

	chat, err := b.Config.Model.ChatByChatID(chatID)
	if err != nil {
		log.Panicf("Error: %v, chatID: %v", err, chatID)
	}

	if chat.Type == "private" {
		statesConfig = b.Config.StatesConfigPrivate
	} else {
		statesConfig = b.Config.StatesConfigGroup
	}

	for {

		if _, ok := statesConfig[chat.State.Name]; !ok {
			log.Panicf("No such state: %v", chat.State.Name)
		}

		while := statesConfig[chat.State.Name].While
		after := statesConfig[chat.State.Name].After
		if while != nil {
		WhileLoop:
			for {
				signal := while(b, signalChan)

				switch signal := signal.(type) {
				case tgbotapi.Update:
					update = signal
					b.updateChat(update, chat)
					b.Config.ChatLog(b, update, Chat(*chat))
					break WhileLoop
				case State:
					chat.State = signal
					log.Printf("    Interrupted with state: %v", chat.State)
					goto BeforeLabel
				case tgbotapi.MessageConfig:
					msg := signal
					msg.ChatID = int64(chat.ChatID)
					_, err := b.api.Send(msg)
					if err != nil {
						log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
						if err.Error() == "forbidden" {
							chat.Abandoned = true
						}
					}
					continue WhileLoop
				case tgbotapi.PhotoConfig:
					msg := signal
					msg.ChatID = int64(chat.ChatID)
					_, err := b.api.Send(msg)
					if err != nil {
						log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
						if err.Error() == "forbidden" {
							chat.Abandoned = true
						}
					}
					continue WhileLoop
				case tgbotapi.DocumentConfig:
					msg := signal
					msg.ChatID = int64(chat.ChatID)
					_, err := b.api.Send(msg)
					if err != nil {
						log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
						if err.Error() == "forbidden" {
							chat.Abandoned = true
						}
					}
					continue WhileLoop
				case tgbotapi.Chattable: // todo: leave only one of Message/PhotoConfig and Chattable
					msg := signal
					// fix ChatID in this message!!
					_, err := b.api.Send(msg)
					if err != nil {
						log.Printf("Failed to send (%v): error \"%v\"\n", marshal(msg), err)
						if err.Error() == "forbidden" {
							chat.Abandoned = true
						}
					}
					continue WhileLoop
				default:
					log.Panicf("Should be either Update, State, Message/PhotoConfig or Chattable")
				}
			}
		}

		// todo: consider chat.Abandoned from here?
		if after != nil {
			after(b, Chat(*chat), update, &chat.State, &chat.Params)

			log.Printf("    State after: %v", chat.State)
		}

	BeforeLabel:
		if !chat.State.skipBefore {
			before := statesConfig[chat.State.Name].Before
			if before != nil {
				before(b, Chat(*chat))
			}
		}

		err = b.Config.Model.Update(chat)
		if err != nil {
			log.Panic(err)
		}
	}
}

// goroutine
func (b Bot) processSendChan() {
	const (
		commonDelay = time.Second / 30
	)

	for chatSignal := range b.SendChan {
		b.sendSignal(chatSignal.ChatID, chatSignal.Signal)
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
			b.sendSignal(chatID, broadSignal.Signal)
			time.Sleep(commonDelay)
		}
	}
}

func (b Bot) sendSignal(chatID ChatID, signal Signal) {
	b.chatsChans.RLock()
	// if b.chatsChans.m[chatID] == nil {
	// 	b.chatsChans.m[chatID] = make(chan Signal, chatChanBufSize)
	// 	go b.processChat(chatID, b.chatsChans.m[chatID])
	// }
	if b.chatsChans.m[chatID] != nil {
		b.chatsChans.m[chatID] <- signal
	}
	b.chatsChans.RUnlock()
}
