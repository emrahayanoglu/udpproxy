package main

import (
	"net"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	Source    string   `long:"source" default:":2203" description:"Source port to listen on"`
	Filter    string   `long:"filter" default:"" description:"Filter only packets if it is not received from IP:Port address specified"`
	SetSource string   `long:"setsource" default:"" description:"Set Source when sending packet to targets"`
	Target    []string `long:"target" description:"Target address to forward to"`
	Quiet     bool     `long:"quiet" description:"whether to print logging info or not"`
	Buffer    int      `long:"buffer" default:"10240" description:"max buffer size for the socket io"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		if !strings.Contains(err.Error(), "Usage") {
			log.Printf("error: %v\n", err.Error())
			os.Exit(1)
		} else {
			// log.Printf("%v\n", err.Error())
			os.Exit(0)
		}
	}

	if opts.Quiet {
		log.SetLevel(log.WarnLevel)
	}

	sourceAddr, err := net.ResolveUDPAddr("udp", opts.Source)
	if err != nil {
		log.WithError(err).Fatal("Could not resolve source address:", opts.Source)
		return
	}

	var targetAddr []*net.UDPAddr
	for _, v := range opts.Target {
		addr, err := net.ResolveUDPAddr("udp", v)
		if err != nil {
			log.WithError(err).Fatal("Could not resolve target address:", v)
			return
		}
		targetAddr = append(targetAddr, addr)
	}

	sourceConn, err := net.ListenUDP("udp", sourceAddr)
	if err != nil {
		log.WithError(err).Fatal("Could not listen on address:", opts.Source)
		return
	}

	var setSource *net.UDPAddr

	if opts.SetSource != "" {
		setSource, err = net.ResolveUDPAddr("udp", opts.SetSource)

		if err != nil {
			log.WithError(err).Fatal("Could not create source address:", opts.SetSource)
			return
		}
	}

	defer sourceConn.Close()

	log.Printf(">> Starting udpproxy, Source at %v, Target at %v...", opts.Source, opts.Target)

	for {
		b := make([]byte, opts.Buffer)
		n, addr, err := sourceConn.ReadFromUDP(b)

		if err != nil {
			log.WithError(err).Error("Could not receive a packet")
			continue
		}

		log.WithField("addr", addr.String()).WithField("bytes", n).Info("Packet received")
		if opts.Filter != "" && addr.String() != opts.Filter {
			log.Info("Packet is filtered out since it is not received from %s", addr.String())
			continue
		}

		for _, v := range targetAddr {
			conn, err := net.DialUDP("udp", setSource, v)
			if err != nil {
				log.WithError(err).Fatal("Could not connect to target address:", v)
				return
			}

			if _, err := conn.Write(b[0:n]); err != nil {
				log.WithError(err).Warn("Could not forward packet.")
			}

			conn.Close()
		}
	}
}
