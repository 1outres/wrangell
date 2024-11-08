package gateway

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type (
	UdpGateway struct {
		client client.Client
	}

	Packet struct {
		msg  uint16
		ip   uint32
		port uint16
	}
)

func (g *UdpGateway) Listen() error {
	ln, err := net.ListenPacket("udp", ":6000")
	if err != nil {
		return err
	}
	defer ln.Close()

	var buf [8]byte
	for {
		ctx, _ := context.WithTimeout(context.TODO(), 5*time.Second)

		_, _, err := ln.ReadFrom(buf[:])
		if err != nil {
			log.Errorf(ctx, "Error reading from UDP: %v", err)
			break
		}

		packet := Packet{}

		reader := bytes.NewReader(buf[:])
		err = binary.Read(reader, binary.LittleEndian, &packet)
		if err != nil {
			log.Errorf(ctx, "Error reading from UDP: %v", err)
			break
		}

		go func() {
			err = g.handle(packet)
			if err != nil {
				log.Errorf(ctx, "Error reading from UDP: %v", err)
			}
		}()
	}

	return nil
}

func (g *UdpGateway) handle(packet Packet) error {
	if packet.msg == 0x9 {
		// TODO: Start Request
	}
}
