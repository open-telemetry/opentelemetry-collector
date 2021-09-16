package myip

import (
	"fmt"
	"net"
	"sync"
)

var once sync.Once

var myIp = []string{}

func Set() {
	once.Do(func() {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			panic(fmt.Errorf("get addr fail;err %s", err))
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				myIp = append(myIp, ipnet.IP.String())
			}
		}
	})
}

func Get() []string {
	return myIp
}
