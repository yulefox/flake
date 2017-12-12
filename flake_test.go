package flake

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"
)

func TestGenID(*testing.T) {
	data, err := ioutil.ReadFile("flake.json")
	if err != nil {
		log.Fatal(err)
	}
	settings := make(map[string]Settings)
	err = json.Unmarshal(data, &settings)
	for t := range settings {
		for i := 0; i < 100; i++ {
			id, _ := GenID(t)
			log.Printf("%s: %x\n", t, id)
		}
	}
}
