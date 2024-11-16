package wrangellpkt

import (
	"encoding/binary"
	"fmt"
	"net"
)

type (
	Packet struct {
		Msg Message // 8 bits
		*HelloPacket
		*TargetPacket
		*ReqPacket
	}

	HelloPacket struct {
		Count uint16 // 16 bits
	}

	TargetPacket struct {
		Ip       uint32 // 32 bits
		Port     uint16 // 16 bits
		Replicas uint16 // 16 bits
	}

	ReqPacket struct {
		Ip   uint32 // 32 bits
		Port uint16 // 16 bits
	}

	Message byte
)

const (
	MessageHello   Message = 0x00
	MessageTarget  Message = 0x01
	MessageRequest Message = 0x02
)

func PacketFromBuffer(buf []byte) *Packet {
	if len(buf) < 1 {
		return nil
	}
	msg := Message(buf[0])
	switch msg {
	case MessageHello:
		if len(buf) < 3 {
			return nil
		}
		return &Packet{
			Msg: msg,
			HelloPacket: &HelloPacket{
				Count: uint16(buf[1]) | uint16(buf[2])<<8,
			},
		}
	case MessageTarget:
		if len(buf) < 9 {
			return nil
		}
		return &Packet{
			Msg: msg,
			TargetPacket: &TargetPacket{
				Ip:       uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24,
				Port:     uint16(buf[5]) | uint16(buf[6])<<8,
				Replicas: uint16(buf[7]) | uint16(buf[8])<<8,
			},
		}
	case MessageRequest:
		if len(buf) < 7 {
			return nil
		}
		return &Packet{
			Msg: msg,
			ReqPacket: &ReqPacket{
				Ip:   uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24,
				Port: uint16(buf[5]) | uint16(buf[6])<<8,
			},
		}
	default:
		return nil
	}
}

func (p *Packet) ToBuffer() []byte {
	switch p.Msg {
	case MessageHello:
		if p.HelloPacket == nil {
			return nil
		}
		buf := make([]byte, 3)
		buf[0] = byte(p.Msg)
		buf[1] = byte(p.HelloPacket.Count)
		buf[2] = byte(p.HelloPacket.Count >> 8)
		return buf
	case MessageTarget:
		if p.TargetPacket == nil {
			return nil
		}
		buf := make([]byte, 9)
		buf[0] = byte(p.Msg)
		buf[1] = byte(p.TargetPacket.Ip)
		buf[2] = byte(p.TargetPacket.Ip >> 8)
		buf[3] = byte(p.TargetPacket.Ip >> 16)
		buf[4] = byte(p.TargetPacket.Ip >> 24)
		buf[5] = byte(p.TargetPacket.Port)
		buf[6] = byte(p.TargetPacket.Port >> 8)
		buf[7] = byte(p.TargetPacket.Replicas)
		buf[8] = byte(p.TargetPacket.Replicas >> 8)
		return buf
	case MessageRequest:
		if p.ReqPacket == nil {
			return nil
		}
		buf := make([]byte, 7)
		buf[0] = byte(p.Msg)
		buf[1] = byte(p.ReqPacket.Ip)
		buf[2] = byte(p.ReqPacket.Ip >> 8)
		buf[3] = byte(p.ReqPacket.Ip >> 16)
		buf[4] = byte(p.ReqPacket.Ip >> 24)
		buf[5] = byte(p.ReqPacket.Port)
		buf[6] = byte(p.ReqPacket.Port >> 8)
		return buf
	default:
		return nil
	}
}

func (p *Packet) String() string {
	switch p.Msg {
	case MessageHello:
		return fmt.Sprintf("{msg: %s, count: %d}", p.Msg, p.HelloPacket.Count)
	case MessageTarget:
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, p.TargetPacket.Ip)
		return fmt.Sprintf("{msg: %s, ip: %s, port: %d, replicas: %d}", p.Msg, ip.String(), p.TargetPacket.Port, p.TargetPacket.Replicas)
	case MessageRequest:
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, p.ReqPacket.Ip)
		return fmt.Sprintf("{msg: %s, ip: %s, port: %d}", p.Msg, ip.String(), p.ReqPacket.Port)
	default:
		return "Unknown"
	}
}

func (msg Message) String() string {
	switch msg {
	case MessageHello:
		return "Hello"
	case MessageTarget:
		return "Target"
	case MessageRequest:
		return "Request"
	default:
		return "Unknown"
	}
}
