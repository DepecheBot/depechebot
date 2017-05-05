// This file contains fixes for tgbotapi library
package depechebot

import (
	"log"
	"time"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// GetUpdatesChan fixes tgbotapi function by adding stop signal channel
func GetUpdatesChan(bot *tgbotapi.BotAPI, config tgbotapi.UpdateConfig) (<-chan tgbotapi.Update, chan<- struct{}, error) {
	updatesChan := make(chan tgbotapi.Update, 100)
	stopChan := make(chan struct{})

	go func() {
		for {
			updates, err := bot.GetUpdates(config)
			if err != nil {
				log.Println(err)
				log.Println("Failed to get updates, retrying in 3 seconds...")
				time.Sleep(time.Second * 3)

				continue
			}

			for _, update := range updates {
				if update.UpdateID >= config.Offset {
					config.Offset = update.UpdateID + 1
					updatesChan <- update
				}
			}

			select {
			case <-stopChan:
				close(updatesChan)
				close(stopChan)
				return
			default:
				continue
			}
		}
	}()

	return updatesChan, stopChan, nil
}
