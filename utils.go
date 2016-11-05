package depechebot

import (
	"encoding/json"
	"log"
)

func marshal(data interface{}) string {
	out, err := json.Marshal(data)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}
