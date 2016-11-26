package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"

	dbot "github.com/depechebot/depechebot"
)

type Model struct {
	db *sql.DB
	//tableName string
	//chats []*dbot.Chat
}

func NewModel(db *sql.DB) Model {
	return Model{db: db}
}

// Init initializes model.
// num is the number of existing chats.
func (m Model) Init() (chatIDs []dbot.ChatID, err error) {

	err = m.db.Ping()
	if err != nil {
		return nil, err
	}

	err = m.createTable()
	if err != nil {
		return nil, err
	}

	const sqlstr = `SELECT chat_id from ` + `chat`
	q, err := m.db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	var chatID dbot.ChatID
	chatIDs = []dbot.ChatID{}
	for q.Next() {
		err = q.Scan(&chatID)
		if err != nil {
			return nil, err
		}

		chatIDs = append(chatIDs, chatID)
	}

	return chatIDs, nil
}

func (m Model) createTable() error {
	var err error

	const sqlstr = `CREATE TABLE IF NOT EXISTS ` +
		`chat` +
		` (
  primary_id INTEGER NOT NULL PRIMARY KEY,
  chat_id BIGINT UNIQUE NOT NULL,
  type TEXT NOT NULL,
  abandoned INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  user_name TEXT NOT NULL DEFAULT '',
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  open_time DATETIME NOT NULL,
  last_time DATETIME NOT NULL,
  state TEXT NOT NULL,
  params TEXT NOT NULL
);
`
	_, err = m.db.Exec(sqlstr)

	return err
}

// Exists determines if the Chat exists in the database.
func (m Model) Exists(c *dbot.Chat) (exists bool, err error) {
	var cnt int
	var sqlstr = `SELECT count(*) as count from ` + `chat` + ` where chat_id = ?`
	err = m.db.QueryRow(sqlstr, c.ChatID).Scan(&cnt)
	return cnt != 0, err
}

// Insert inserts chat to the database.
// Sets c.PrimaryID.
func (m Model) Insert(c *dbot.Chat) error {
	var err error

	const sqlstr = `INSERT INTO chat (` +
		`chat_id, type, abandoned, user_id, user_name, first_name, last_name, open_time, last_time, state, params` +
		`) VALUES (` +
		`?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?` +
		`)`

	state, err := json.Marshal(c.State)
	if err != nil {
		return err
	}
	params, err := json.Marshal(c.Params)
	if err != nil {
		return err
	}

	res, err := m.db.Exec(sqlstr, c.ChatID, c.Type, c.Abandoned, c.UserID, c.UserName,
		c.FirstName, c.LastName, c.OpenTime, c.LastTime, string(state), string(params))
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	c.PrimaryID = int(id)

	return nil
}

// Update updates the Chat in the database.
func (m Model) Update(c *dbot.Chat) error {
	var err error

	const sqlstr = `UPDATE chat SET ` +
		`primary_id = ?, type = ?, abandoned = ?, user_id = ?, user_name = ?, first_name = ?, last_name = ?, open_time = ?, last_time = ?, state = ?, params = ?` +
		` WHERE chat_id = ?`

	state, err := json.Marshal(c.State)
	if err != nil {
		return err
	}

	params, err := json.Marshal(c.Params)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(sqlstr, c.PrimaryID, c.Type, c.Abandoned, c.UserID, c.UserName,
		c.FirstName, c.LastName, c.OpenTime, c.LastTime, string(state), string(params), c.ChatID)
	return err
}

// Save saves the Chat to the database.
// Prefer Update() if you know that chat exists.
func (m Model) Save(c *dbot.Chat) error {
	exists, err := m.Exists(c)
	if err != nil {
		return err
	}
	if exists {
		return m.Update(c)
	}

	return m.Insert(c)
}

// Delete deletes the Chat from the database.
func (m Model) Delete(c *dbot.Chat) error {
	var err error

	const sqlstr = `DELETE FROM chat WHERE chat_id = ?`

	_, err = m.db.Exec(sqlstr, c.ChatID)
	return err
}

// ChatByPrimaryID retrieves a chat by primaryID.
func (m Model) ChatByPrimaryID(primaryID int) (*dbot.Chat, error) {
	var err error
	var state, params string

	const sqlstr = `SELECT ` +
		`primary_id, chat_id, type, abandoned, user_id, user_name, first_name, last_name, open_time, last_time, state, params ` +
		`FROM chat ` +
		`WHERE primary_id = ?`

	c := dbot.Chat{}
	err = m.db.QueryRow(sqlstr, primaryID).Scan(&c.PrimaryID, &c.ChatID, &c.Type, &c.Abandoned, &c.UserID, &c.UserName,
		&c.FirstName, &c.LastName, &c.OpenTime, &c.LastTime, &state, &params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("chat not found (use Model.Exist() before if not sure)")
		} else {
			return nil, err
		}
	}

	err = json.Unmarshal([]byte(state), &c.State)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(params), &c.Params)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChatByChatID retrieves a chat by chatID.
func (m Model) ChatByChatID(chatID dbot.ChatID) (*dbot.Chat, error) {
	var err error
	var state, params string

	const sqlstr = `SELECT ` +
		`primary_id, chat_id, type, abandoned, user_id, user_name, first_name, last_name, open_time, last_time, state, params ` +
		`FROM chat ` +
		`WHERE chat_id = ?`

	c := dbot.Chat{}
	err = m.db.QueryRow(sqlstr, chatID).Scan(&c.PrimaryID, &c.ChatID, &c.Type, &c.Abandoned, &c.UserID, &c.UserName,
		&c.FirstName, &c.LastName, &c.OpenTime, &c.LastTime, &state, &params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("chat not found (use Model.Exist() before if not sure)")
		} else {
			return nil, err
		}
	}

	err = json.Unmarshal([]byte(state), &c.State)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(params), &c.Params)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChatsByParam retrieves chats with chat.Params matching param.
func (m Model) ChatsByParam(param string) ([]*dbot.Chat, error) {
	var err error
	var state, params string

	const sqlstr = `SELECT ` +
		`primary_id, chat_id, type, abandoned, user_id, user_name, first_name, last_name, open_time, last_time, state, params ` +
		`FROM chat ` +
		`WHERE ` +
		`params like "%" || ? || "%"`

	q, err := m.db.Query(sqlstr, param)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*dbot.Chat{}, nil
		} else {
			return nil, err
		}
	}
	defer q.Close()

	chats := []*dbot.Chat{}
	for q.Next() {
		c := dbot.Chat{}

		err = q.Scan(&c.PrimaryID, &c.ChatID, &c.Type, &c.Abandoned, &c.UserID, &c.UserName,
			&c.FirstName, &c.LastName, &c.OpenTime, &c.LastTime, &state, &params)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(state), &c.State)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(params), &c.Params)
		if err != nil {
			return nil, err
		}

		chats = append(chats, &c)
	}

	return chats, nil
}
