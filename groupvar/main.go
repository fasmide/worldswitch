package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Response []Endpoint

type Endpoint struct {
	Hostname string
	Country  string `json:"country_name"`
	City     string `json:"city_name"`
	Active   bool
	IPv4     string `json:"ipv4_addr_in"`
	Type     string
	PubKey   string
}

type GroupVars struct {
	Order   []string         `yaml:"order"`
	Routing map[int]Endpoint `yaml:"routing"`
}

func main() {
	endpoints, err := fetchEndpoints()
	if err != nil {
		log.Fatal(err)
	}

	// make sure the endpoints are in no particular order
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(endpoints), func(i, j int) { endpoints[i], endpoints[j] = endpoints[j], endpoints[i] })

	groupVars, err := loadGroupVars()
	if err != nil {
		log.Fatal(err)
	}

	for i, name := range groupVars.Order {
		var found bool
		for _, candidate := range endpoints {
			if !candidate.Active {
				continue
			}

			if candidate.Type != "wireguard" {
				continue
			}

			if candidate.City == name {
				groupVars.Routing[i] = candidate
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("unable to find candidate for %s", name)
		}
	}

	fd, err := os.Create("../ansible/group_vars/all.yml")
	if err != nil {
		log.Fatalf("unable to open \"../ansible/group_vars/all.yml\" for writing: %s", err)
	}

	enc := yaml.NewEncoder(fd)
	err = enc.Encode(groupVars)
	if err != nil {
		log.Fatalf("unable to write data: %s", err)
	}

	log.Printf("groupvars updated...")

}

func loadGroupVars() (GroupVars, error) {
	fd, err := os.Open("../ansible/group_vars/all.yml")
	if err != nil {
		return GroupVars{}, fmt.Errorf("unable to open group_vars/all.yml: %w", err)
	}
	defer fd.Close()

	g := GroupVars{Routing: make(map[int]Endpoint)}
	dec := yaml.NewDecoder(fd)
	err = dec.Decode(&g)
	if err != nil {
		return GroupVars{}, fmt.Errorf("unable to decode groupvars: %w", err)
	}
	return g, nil
}

func fetchEndpoints() (Response, error) {
	resp, err := http.Get("https://api.mullvad.net/www/relays/all/")
	if err != nil {
		return nil, fmt.Errorf("unable to fetch resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected statuscode: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)

	var endpoints Response
	err = dec.Decode(&endpoints)
	if err != nil {
		return nil, fmt.Errorf("unable to decode json: %w", err)
	}

	return endpoints, nil
}
