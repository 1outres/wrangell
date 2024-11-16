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
		SrcIp   uint32 // 32 bits
		SrcPort uint16 // 16 bits
		DstIp   uint32 // 32 bits
		DstPort uint16 // 16 bits
		Seq     uint32 // 32 bits
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
		if len(buf) < 17 {
			return nil
		}
		return &Packet{
			Msg: msg,
			ReqPacket: &ReqPacket{
				DstIp:   uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24,
				DstPort: uint16(buf[5]) | uint16(buf[6])<<8,
				Seq:     uint32(buf[7]) | uint32(buf[8])<<8 | uint32(buf[9])<<16 | uint32(buf[10])<<24,
				SrcIp:   uint32(buf[11]) | uint32(buf[12])<<8 | uint32(buf[13])<<16 | uint32(buf[14])<<24,
				SrcPort: uint16(buf[15]) | uint16(buf[16])<<8,
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
		buf := make([]byte, 17)
		buf[0] = byte(p.Msg)
		buf[1] = byte(p.ReqPacket.DstIp)
		buf[2] = byte(p.ReqPacket.DstIp >> 8)
		buf[3] = byte(p.ReqPacket.DstIp >> 16)
		buf[4] = byte(p.ReqPacket.DstIp >> 24)
		buf[5] = byte(p.ReqPacket.DstPort)
		buf[6] = byte(p.ReqPacket.DstPort >> 8)
		buf[7] = byte(p.ReqPacket.Seq)
		buf[8] = byte(p.ReqPacket.Seq >> 8)
		buf[9] = byte(p.ReqPacket.Seq >> 16)
		buf[10] = byte(p.ReqPacket.Seq >> 24)
		buf[11] = byte(p.ReqPacket.SrcIp)
		buf[12] = byte(p.ReqPacket.SrcIp >> 8)
		buf[13] = byte(p.ReqPacket.SrcIp >> 16)
		buf[14] = byte(p.ReqPacket.SrcIp >> 24)
		buf[15] = byte(p.ReqPacket.SrcPort)
		buf[16] = byte(p.ReqPacket.SrcPort >> 8)
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
		dstip := make(net.IP, 4)
		binary.BigEndian.PutUint32(dstip, p.ReqPacket.DstIp)
		srcip := make(net.IP, 4)
		binary.BigEndian.PutUint32(srcip, p.ReqPacket.SrcIp)
		return fmt.Sprintf("{msg: %s, dstip: %s, dstport: %d, seq: %d, srcip %s, srcport %d}", p.Msg, dstip.String(), p.ReqPacket.DstPort, p.ReqPacket.Seq, srcip.String(), p.ReqPacket.SrcPort)
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
