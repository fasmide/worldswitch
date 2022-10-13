package main

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// hardcoding our way through
var world = []string{
	"you will never see this", // e.g. index zero - ports start with 1
	"Melbourne",
	"Sydney",
	"Vienna",
	"Brussels",
	"Sao Paulo",
	"Sofia",
	"Toronto",
	"Vancouver",
	"Prague",
	"Copenhagen",
	"Tallinn",
	"Helsinki",
	"Marseille",
	"Paris",
	"Dusseldorf",
	"Frankfurt",
	"Hong Kong",
	"Budapest",
	"Dublin",
	"Tel Aviv",
	"Milan",
	"Osaka",
	"Tokyo",
	"Riga",
	"Luxembourg",
	"Chisinau",
	"Amsterdam",
	"Auckland",
	"Skopje",
	"Oslo",
	"Warsaw",
	"Lisbon",
	"Bucharest",
	"Belgrade",
	"Singapore",
	"Bratislava",
	"Madrid",
	"Gothenburg",
	"Stockholm",
	"Zurich",
	"London",
	"Manchester",
	"Ashburn, VA",
	"Chicago, IL",
	"Houston, TX",
	"Los Angeles, CA",
	"Dark Webs, Tor",
	"welcome to nothing", // 48
}

func main() {
	r := NewRenew("eth0")
	err := netlink.LinkSubscribe(r.Changes, r.Done)
	if err != nil {
		panic(err)
	}

	s := NewSSID()
	err = netlink.AddrSubscribe(s.Changes, s.Done)
	if err != nil {
		panic(err)
	}

	log.Printf("We should be setup")
	select {}
}

type SSID struct {
	Changes chan netlink.AddrUpdate
	Done    chan struct{}

	CurrentIndex int
}

func NewSSID() *SSID {
	ssid := &SSID{
		Changes: make(chan netlink.AddrUpdate),
		Done:    make(chan struct{}),
	}

	go ssid.loop()
	return ssid
}

func (s *SSID) loop() {
	for {
		c := <-s.Changes
		if c.NewAddr {
			if c.LinkAddress.IP.To4() == nil {
				continue
			}

			log.Printf("we have a new Addr: %+v", c)

			// this is not going to be pretty
			// we could of cause use come cidr Contain instead but hey...
			var one, two, tree, four int
			_, err := fmt.Sscanf(c.LinkAddress.IP.String(), "%d.%d.%d.%d", &one, &two, &tree, &four)
			if err != nil {
				panic(err)
			}
			log.Printf("we are at the \"%s\" network", world[tree])

			if s.CurrentIndex == tree {
				log.Printf("nothing to do here...")
				continue
			}

			s.CurrentIndex = tree
			start := time.Now()

			cmd := exec.Command("uci", "set", fmt.Sprintf("wireless.wifinet0.ssid=%s", world[tree]))
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("unable to run command: %s\n%s", err, string(out))
			}

			cmd = exec.Command("uci", "commit")
			out, err = cmd.CombinedOutput()
			if err != nil {
				log.Printf("unable to run command: %s\n%s", err, string(out))
			}

			cmd = exec.Command("wifi", "reload")
			out, err = cmd.CombinedOutput()
			if err != nil {
				log.Printf("unable to run command: %s\n%s", err, string(out))
			}

			log.Printf("reload took %s", time.Since(start))
		}
	}
}

type Renew struct {
	IfName string
	State  string

	Changes chan netlink.LinkUpdate
	Done    chan struct{}
}

func NewRenew(s string) *Renew {
	r := &Renew{
		IfName:  s,
		Changes: make(chan netlink.LinkUpdate),
		Done:    make(chan struct{}),
	}

	go r.loop()
	return r
}

func (e *Renew) loop() {
	for {
		c := <-e.Changes
		a := c.Link.Attrs()
		log.Printf("link change: %s: %s", a.Name, a.OperState)
		state := a.OperState.String()
		if a.Name != e.IfName {
			continue
		}

		if e.State == state {
			continue
		}

		// this is the transition we are looking for
		if e.State == "down" && state == "up" {
			e.Do()
		}

		e.State = state
	}

}
func (e *Renew) Do() {
	log.Printf("trying to renew lease, looking up process udhcpc")
	pids, err := process.Processes()
	if err != nil {
		panic(err)
	}
	for _, v := range pids {
		n, _ := v.Name()
		if n == "udhcpc" {
			log.Printf("signaling %s (%d)", n, v.Pid)
			// SIGUSR2	throw away current lease
			err := v.SendSignal(unix.SIGUSR2)
			if err != nil {
				panic(err)
			}

			// USR1 try to obtain new lease
			// since we did the USR2 first, there is no "renew" process going on
			// and udhcpc wont try to contact the old dhcpd
			err = v.SendSignal(unix.SIGUSR1)
			if err != nil {
				panic(err)
			}
			break
		}
	}
}
