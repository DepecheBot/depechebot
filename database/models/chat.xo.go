// Package models contains the types for schema ''.
package models

// GENERATED BY XO. DO NOT EDIT.

import "errors"

// Chat represents a row from 'chat'.
type Chat struct {
	PrimaryID int    `json:"primary_id"` // primary_id
	ChatID    int    `json:"chat_id"`    // chat_id
	Type      string `json:"type"`       // type
	Abandoned int    `json:"abandoned"`  // abandoned
	UserID    int    `json:"user_id"`    // user_id
	UserName  string `json:"user_name"`  // user_name
	RealName  string `json:"real_name"`  // real_name
	FirstName string `json:"first_name"` // first_name
	LastName  string `json:"last_name"`  // last_name
	OpenTime  string `json:"open_time"`  // open_time
	LastTime  string `json:"last_time"`  // last_time
	Groups    string `json:"groups"`     // groups
	State     string `json:"state"`      // state

	// xo fields
	_exists, _deleted bool
}

// Exists determines if the Chat exists in the database.
func (c *Chat) Exists() bool {
	return c._exists
}

// Deleted provides information if the Chat has been deleted from the database.
func (c *Chat) Deleted() bool {
	return c._deleted
}

// Insert inserts the Chat to the database.
func (c *Chat) Insert(db XODB) error {
	var err error

	// if already exist, bail
	if c._exists {
		return errors.New("insert failed: already exists")
	}

	// sql query
	const sqlstr = `INSERT INTO chat (` +
		`chat_id, type, abandoned, user_id, user_name, real_name, first_name, last_name, open_time, last_time, groups, state` +
		`) VALUES (` +
		`?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?` +
		`)`

	// run query
	XOLog(sqlstr, c.ChatID, c.Type, c.Abandoned, c.UserID, c.UserName, c.RealName, c.FirstName, c.LastName, c.OpenTime, c.LastTime, c.Groups, c.State)
	res, err := db.Exec(sqlstr, c.ChatID, c.Type, c.Abandoned, c.UserID, c.UserName, c.RealName, c.FirstName, c.LastName, c.OpenTime, c.LastTime, c.Groups, c.State)
	if err != nil {
		return err
	}

	// retrieve id
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// set primary key and existence
	c.PrimaryID = int(id)
	c._exists = true

	return nil
}

// Update updates the Chat in the database.
func (c *Chat) Update(db XODB) error {
	var err error

	// if doesn't exist, bail
	if !c._exists {
		return errors.New("update failed: does not exist")
	}

	// if deleted, bail
	if c._deleted {
		return errors.New("update failed: marked for deletion")
	}

	// sql query
	const sqlstr = `UPDATE chat SET ` +
		`chat_id = ?, type = ?, abandoned = ?, user_id = ?, user_name = ?, real_name = ?, first_name = ?, last_name = ?, open_time = ?, last_time = ?, groups = ?, state = ?` +
		` WHERE primary_id = ?`

	// run query
	XOLog(sqlstr, c.ChatID, c.Type, c.Abandoned, c.UserID, c.UserName, c.RealName, c.FirstName, c.LastName, c.OpenTime, c.LastTime, c.Groups, c.State, c.PrimaryID)
	_, err = db.Exec(sqlstr, c.ChatID, c.Type, c.Abandoned, c.UserID, c.UserName, c.RealName, c.FirstName, c.LastName, c.OpenTime, c.LastTime, c.Groups, c.State, c.PrimaryID)
	return err
}

// Save saves the Chat to the database.
func (c *Chat) Save(db XODB) error {
	if c.Exists() {
		return c.Update(db)
	}

	return c.Insert(db)
}

// Delete deletes the Chat from the database.
func (c *Chat) Delete(db XODB) error {
	var err error

	// if doesn't exist, bail
	if !c._exists {
		return nil
	}

	// if deleted, bail
	if c._deleted {
		return nil
	}

	// sql query
	const sqlstr = `DELETE FROM chat WHERE primary_id = ?`

	// run query
	XOLog(sqlstr, c.PrimaryID)
	_, err = db.Exec(sqlstr, c.PrimaryID)
	if err != nil {
		return err
	}

	// set deleted
	c._deleted = true

	return nil
}

// ChatByPrimaryID retrieves a row from 'chat' as a Chat.
//
// Generated from index 'chat_primary_id_pkey'.
func ChatByPrimaryID(db XODB, primaryID int) (*Chat, error) {
	var err error

	// sql query
	const sqlstr = `SELECT ` +
		`primary_id, chat_id, type, abandoned, user_id, user_name, real_name, first_name, last_name, open_time, last_time, groups, state ` +
		`FROM chat ` +
		`WHERE primary_id = ?`

	// run query
	XOLog(sqlstr, primaryID)
	c := Chat{
		_exists: true,
	}

	err = db.QueryRow(sqlstr, primaryID).Scan(&c.PrimaryID, &c.ChatID, &c.Type, &c.Abandoned, &c.UserID, &c.UserName, &c.RealName, &c.FirstName, &c.LastName, &c.OpenTime, &c.LastTime, &c.Groups, &c.State)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChatByChatID retrieves a row from 'chat' as a Chat.
//
// Generated from index 'sqlite_autoindex_chat_1'.
func ChatByChatID(db XODB, chatID int) (*Chat, error) {
	var err error

	// sql query
	const sqlstr = `SELECT ` +
		`primary_id, chat_id, type, abandoned, user_id, user_name, real_name, first_name, last_name, open_time, last_time, groups, state ` +
		`FROM chat ` +
		`WHERE chat_id = ?`

	// run query
	XOLog(sqlstr, chatID)
	c := Chat{
		_exists: true,
	}

	err = db.QueryRow(sqlstr, chatID).Scan(&c.PrimaryID, &c.ChatID, &c.Type, &c.Abandoned, &c.UserID, &c.UserName, &c.RealName, &c.FirstName, &c.LastName, &c.OpenTime, &c.LastTime, &c.Groups, &c.State)
	if err != nil {
		return nil, err
	}

	return &c, nil
}