package database

import (
	"database/sql"

	"github.com/depechebot/depechebot/database/models"
)

// type State struct {
// 	state int
// }

//type Chat models.Chat

// type Chat struct {
// 	PrimaryID int

// 	// Telegram info
// 	ChatID    ChatIDType
// 	UserID    int64
// 	Username  string
// 	Firstname string
// 	Lastname  string

// 	// datetimes
// 	OpenTime time.Time
// 	LastTime  time.Time

// 	// state
// 	State

// 	// parameters
// 	Realname string
// 	Groups   string
// }

var Chats []*models.Chat

func LoadChatsFromDB() error {
	primaryID := 1
	for {
		chat, err := models.ChatByPrimaryID(DB, primaryID)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		Chats = append(Chats, chat)
		primaryID += 1
	}
}

//writeToDB()
