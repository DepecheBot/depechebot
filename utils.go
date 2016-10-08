package depechebot

import (
	"log"
	"encoding/json"
)

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}

func int2bool(i int) bool {
	return i != 0
}

func marshal(data interface{}) string {
	out, err := json.Marshal(data)
	check(err)
	return string(out)
}
