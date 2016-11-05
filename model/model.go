// package model represents Model for depechebot chats data
package model

import "time"

// Model of depechebot data.
type Model interface {
	// Init initializes model.
	// num is the number of existing chats.
	Init() (num int, err error)

	Exists(*Chat) (bool, error)
	Insert(*Chat) error
	Update(*Chat) error
	Save(*Chat) error
	Delete(*Chat) error

	ChatByPrimaryID(id int) (*Chat, error)
	ChatByChatID(id int64) (*Chat, error)
	ChatsByParam(param string) ([]*Chat, error)
}

// Chat represents a row from 'chat'.
type Chat struct {
	PrimaryID int       `json:"primary_id"`
	ChatID    int64     `json:"chat_id"`
	Type      string    `json:"type"`
	Abandoned bool      `json:"abandoned"`
	UserID    int       `json:"user_id"`
	UserName  string    `json:"user_name"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	OpenTime  time.Time `json:"open_time"`
	LastTime  time.Time `json:"last_time"`
	State     State     `json:"state"`
	Params    Params    `json:"params"`
}

// Params
type Params map[string]string

// StateName
type StateName string

// State
type State struct {
	Name       StateName `json:"name"`
	Params     Params    `json:"params"`
	skipBefore bool
}
