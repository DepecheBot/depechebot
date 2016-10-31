package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dataSourceName string) {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Panic(err)
	}

	if err = DB.Ping(); err != nil {
		log.Panic(err)
	}
}

// reads to db.Chats
// func ReadDB() {
// 	rows, err := db.Query("SELECT (chat_id, user_id) FROM chats;")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var chat Chat
// 		if err := rows.Scan(&Chat.ChatID, &Chat.UserID); err != nil {
// 			log.Fatal(err)
// 		}
// 		fmt.Printf("%v, %v\n", Chat.ChatID, Chat.UserID)
// 	}
// 	if err := rows.Err(); err != nil {
// 		log.Fatal(err)
// 	}
// }
