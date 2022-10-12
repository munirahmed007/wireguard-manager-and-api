package network

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/spf13/viper"
	"github.com/vishvananda/netlink"
	"gitlab.com/raspberry.tech/wireguard-manager-and-api/src/db"
)

func SetupWG() {
	log.Println("Info - Setting up WG interface")
	db.WGStart()
	wg0, errLink := netlink.LinkByName("wg0")
	if errLink != nil {
		log.Fatal("Error - Failed to get link to wireguard interface")
	}
	ipCheck(wg0)
}

func addPostWGUp() {
	log.Printf("adding routes")
	addRoutes()
}

func execute(app string, args...string) {
	exec.Command(app, args...)
}

func addRoutes() {
	peers := db.AllPeers()
	app := "ip"
	arg0 := "route"
	arg1 := "add"
	arg2 := "192.168.1.1"
	arg3 := "dev"
	arg4 := "wg0"

	for i, s := range peers {
		if len(s.AllowedIPs) > 0 {
			fmt.Println(i, s.AllowedIPs[0].IP.String())
			arg2 = s.AllowedIPs[0].IP.String() + "/32"
			log.Printf("adding routes ", arg2)
			exec.Command(app, arg0, arg1, arg2, arg3, arg4)
		}
	}
}

func addIP(instance netlink.Link, ipAddr *netlink.Addr) {
	ipAddErr := netlink.AddrAdd(instance, ipAddr)
	if ipAddErr != nil {
		fmt.Println("Warning - Failed to add IP address", ipAddErr)
		log.Println("Warning - Failed to add IP address", ipAddErr)
	} else {
		log.Println("Info - Added IP address to interface")
	}
}

func ipCheck(wg0 netlink.Link) {
	log.Println("Info - Checking if IPs exist")

	IPs, err := netlink.AddrList(wg0, 0) //list of IP addresses in system, equivalent to: `ip addr show`
	if err != nil {
		fmt.Println("Error - Failed to get find wireguard interface")
		log.Fatal("Error - Failed to get find wireguard interface")
	}

	ipv4Check := false //variables for checks
	ipv6Check := false

	wgIPv4 := viper.GetString("INSTANCE.IP.LOCAL.IPV4.ADDRESS") //IPv4 in config
	ipv4Subnet := viper.GetString("INSTANCE.IP.LOCAL.IPV4.SUBNET")
	wgIPv6 := viper.GetString("INSTANCE.IP.LOCAL.IPV6.ADDRESS") //IPv6 in config

	IPv4Addresses := viper.GetStringSlice("INSTANCE.IP.GLOBAL.ADDRESS.IPV4")
	IPv6Addresses := viper.GetStringSlice("INSTANCE.IP.GLOBAL.ADDRESS.IPV6")

	ethDevice := viper.GetString("SERVER.INTERFACE")
	devInterface, errLink := netlink.LinkByName(ethDevice)
	if errLink != nil {
		log.Fatal("Error - Failed to get link to device interface")
	}

	for i := 0; i < len(IPv4Addresses); i++ {
		ipv4AddrParse, errParsev4 := netlink.ParseAddr(IPv4Addresses[i] + "/32") //add subnet of 16 to IP
		if errParsev4 != nil {
			log.Fatal("Error - Failed to parse IPv4 Address")
		}
		addIP(devInterface, ipv4AddrParse)
	}

	for i := 0; i < len(IPv6Addresses); i++ {
		ipv6AddrParse, errParsev6 := netlink.ParseAddr(IPv6Addresses[i] + "/128") //add subnet of 16 to IP
		if errParsev6 != nil {
			log.Fatal("Error - Failed to get parse IPv6 Address")
		}
		addIP(devInterface, ipv6AddrParse)
	}

	ipv4Addr, errParsev4 := netlink.ParseAddr(wgIPv4 + ipv4Subnet) //add subnet of 16 to IP
	if errParsev4 != nil {
		log.Println("Error - Failed to parse IPv4 Address")
	}

	if wgIPv6 != "-" { //if IPv6 is not set to - in config
		ipv6Subnet := viper.GetString("INSTANCE.IP.LOCAL.IPV6.SUBNET")
		ipv6Addr, errParsev6 := netlink.ParseAddr(wgIPv6 + ipv6Subnet)
		if errParsev6 != nil {
			log.Println("Error - Failed to parse IPv6 Address")
		}
		for i := 0; i < len(IPs); i++ { //checks if IPs wanted exist
			if IPs[i].Equal(*ipv4Addr) { //Check if IPv4 address wanted is already present
				ipv4Check = true
			} else if IPs[i].Equal(*ipv6Addr) { //Check if IPv6 address wanted is already present
				ipv6Check = true
			}
		}
		if !ipv6Check {
			addIP(wg0, ipv6Addr) //add IPv6 to system
		}
	} else {
		for i := 0; i < len(IPs); i++ { //checks if IPs wanted exist
			if IPs[i].Equal(*ipv4Addr) { //Check if IPv4 address wanted is already present
				ipv4Check = true
			}
		}
	}
	if !ipv4Check {
		addIP(wg0, ipv4Addr) //add IPv4 to system
	}
	addPostWGUp()
}
