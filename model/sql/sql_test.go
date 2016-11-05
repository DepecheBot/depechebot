package sql

import (
	"database/sql"
	"testing"
	"time"

	"github.com/depechebot/depechebot/model"
	_ "github.com/mattn/go-sqlite3"
)

func TestSqlite3ModelInit(t *testing.T) {
	var m model.Model

	db, err := sql.Open("sqlite3", "./test.sqlite3")
	if err != nil {
		t.Error(err)
	}

	m = NewModel(db)
	num, err := m.Init()
	if err != nil {
		t.Error(err)
	}
	t.Logf("Number of loaded chats: %d", num)
}

func TestSqlite3ModelSaveRetrieveDelete(t *testing.T) {
	var m model.Model

	db, err := sql.Open("sqlite3", "./test2.sqlite3")
	if err != nil {
		t.Error(err)
	}

	m = NewModel(db)
	num, err := m.Init()
	if err != nil {
		t.Error(err)
	}
	t.Logf("Number of loaded chats: %d", num)

	params := model.Params(map[string]string{"foo": "bar", "noo": "bab"})
	state := model.State{Name: "TestState", Params: params}
	chat := &model.Chat{
		ChatID:    88000111222,
		Abandoned: false,
		Type:      "private",
		UserID:    123232,
		UserName:  "username",
		FirstName: "first_name",
		LastName:  "Вава",
		OpenTime:  time.Now(),
		LastTime:  time.Now(),
		State:     state,
		Params:    params,
	}

	err = m.Save(chat)
	if err != nil {
		t.Error(err)
	}

	chat.ChatID = -1324
	exists, err := m.Exists(chat)
	if err != nil {
		t.Error(err)
	}

	if exists {
		chat, err = m.ChatByChatID(chat.ChatID)
		if err != nil {
			t.Error(err)
		}
		chat.Abandoned = !chat.Abandoned
	}

	err = m.Save(chat)
	if err != nil {
		t.Error(err)
	}

	chat2, err := m.ChatByChatID(88000111222)
	if err != nil {
		t.Error(err)
	}
	if chat2.LastName != "Вава" {
		t.Error("ChatByChatID() failed")
	}
	if chat2.State.Params["noo"] != "bab" {
		t.Error("ChatByChatID() failed")
	}

	chat3, err := m.ChatByPrimaryID(chat.PrimaryID)
	if err != nil {
		t.Error(err)
	}
	if chat3.ChatID != -1324 {
		t.Error("ChatByPrimaryID() failed")
	}
	if chat2.State.Params["noo"] != "bab" {
		t.Error("ChatByPrimaryID() failed")
	}

	chats, err := m.ChatsByParam("noo")
	if err != nil {
		t.Error(err)
	}
	if len(chats) != 2 {
		t.Error("ChatsByParam() failed")
	}

	chats[1].LastTime = time.Now()
	err = m.Save(chats[1])
	if err != nil {
		t.Error(err)
	}

	chatID := chats[0].ChatID
	err = m.Delete(chats[0])
	if err != nil {
		t.Error(err)
	}
	_, err = m.ChatByChatID(chatID)
	if err == nil {
		t.Error("Deleted chat retrieved with no error!")
	}

}
