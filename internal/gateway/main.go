package gateway

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"

	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
	"github.com/1outres/wrangell/pkg/wrangellpkt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	Server interface {
		Serve() error
		UpdateTarget(namespace string, name string, ip net.IP, port uint16, replicas uint16)
		Close()
	}

	server struct {
		kubeClient client.Client
		conn       net.PacketConn
		targets    map[[6]byte]wrangell
		address    string
		clients    []net.Conn
	}

	wrangell struct {
		replicas  uint16
		namespace string
		name      string
	}
)

func (s *server) Close() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	if s.clients != nil {
		for _, c := range s.clients {
			if c != nil {
				c.Close()
			}
		}
		s.clients = nil
	}
}

func (s *server) Serve() error {
	listenConfig := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) (err error) {
			return c.Control(func(fd uintptr) {
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if err != nil {
					return
				}
			})
		},
	}
	conn, err := listenConfig.ListenPacket(context.Background(), "udp", s.address)
	if err != nil {
		return err
	}
	s.conn = conn

	var buf [16]byte
	for {
		n, addr, err := conn.ReadFrom(buf[:])
		if err != nil {
			log.Print(err)
			return err
		}

		pkt := wrangellpkt.PacketFromBuffer(buf[:n])

		if pkt.Msg == wrangellpkt.MessageHello {
			log.Printf("client joined : %v", addr)
			d := net.Dialer{
				LocalAddr: conn.LocalAddr(),
				Control: func(network, address string, c syscall.RawConn) (err error) {
					return c.Control(func(fd uintptr) {
						err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
						if err != nil {
							return
						}
					})
				},
			}
			client, err := d.Dial(addr.Network(), addr.String())
			if err != nil {
				log.Print(err)
				continue
			}
			i := len(s.clients)
			s.clients = append(s.clients, client)

			go s.clientRead(i)
		} else {
			log.Print("invalid message received")
		}
	}
}

func (s *server) clientRead(i int) {
	conn := s.clients[i]
	defer func() {
		conn.Close()
		s.clients[i] = nil
	}()

	err := s.sendHello(conn)
	if err != nil {
		log.Printf("failed to send hello packet : %v", conn.RemoteAddr())
		return
	}

	var buf [16]byte
	for {
		conn.SetDeadline(time.Now().Add(90 * time.Second))

		n, err := conn.Read(buf[:])
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				log.Printf("connection timeout : %v", conn.RemoteAddr())
				break
			}
			log.Print(err)
			break
		}

		pkt := wrangellpkt.PacketFromBuffer(buf[:n])
		log.Printf("received from %v, %s", conn.RemoteAddr(), pkt)

		if pkt.Msg == wrangellpkt.MessageHello {
			count := s.count()

			if pkt.HelloPacket.Count != count {
				for ipport, w := range s.targets {
					ip := (net.IP)([]byte{ipport[0], ipport[1], ipport[2], ipport[3]})

					pkt := wrangellpkt.Packet{
						Msg: wrangellpkt.MessageTarget,
						TargetPacket: &wrangellpkt.TargetPacket{
							Ip:       binary.BigEndian.Uint32(ip.To4()),
							Port:     uint16(ipport[4]) | uint16(ipport[5])<<8,
							Replicas: w.replicas,
						},
					}
					_, err := conn.Write(pkt.ToBuffer())
					if err != nil {
						log.Printf("failed to send target packet : %v", conn.RemoteAddr())
						break
					}
				}
			}

			err := s.sendHello(conn)
			if err != nil {
				log.Printf("failed to send hello packet : %v", conn.RemoteAddr())
				break
			}

			continue
		} else if pkt.Msg == wrangellpkt.MessageRequest {
			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, pkt.ReqPacket.Ip)

			fmt.Println(ip.String())

			w, ok := s.targets[[6]byte{ip[0], ip[1], ip[2], ip[3], byte(pkt.ReqPacket.Port), byte(pkt.ReqPacket.Port >> 8)}]

			if ok {
				go func() {
					ctx := context.Background()
					var wrangell wrangellv1alpha1.WrangellService
					err := s.kubeClient.Get(ctx, types.NamespacedName{Namespace: w.namespace, Name: w.name}, &wrangell)
					if err != nil {
						log.Printf("Failed to find WrangellService: %s in %s", w.name, w.namespace)
					}

					wrangell.Status.LatestRequest = metav1.NewTime(time.Now())

					err = s.kubeClient.Status().Update(ctx, &wrangell)
					if err != nil {
						log.Printf("Failed to update WrangellService: %s in %s", w.name, w.namespace)
					}
				}()
			} else {
				log.Printf("target not found : %v", conn.RemoteAddr())
			}
		} else {
			log.Printf("invalid message received: %v", conn.RemoteAddr())
			continue
		}
	}
}

func (s *server) count() uint16 {
	var count uint16 = 0
	for _, w := range s.targets {
		if w.replicas == 0 {
			count++
		}
	}
	return count
}

func (s *server) sendHello(conn net.Conn) error {
	helloPacket := wrangellpkt.Packet{
		Msg: wrangellpkt.MessageHello,
		HelloPacket: &wrangellpkt.HelloPacket{
			Count: s.count(),
		},
	}

	_, err := conn.Write(helloPacket.ToBuffer())
	if err != nil {
		return err
	}
	return nil
}

func (s *server) UpdateTarget(namespace string, name string, ip net.IP, port uint16, replicas uint16) {
	ip = ip.To4()

	pkt := wrangellpkt.Packet{
		Msg: wrangellpkt.MessageTarget,
		TargetPacket: &wrangellpkt.TargetPacket{
			Ip:       binary.BigEndian.Uint32(ip),
			Port:     port,
			Replicas: replicas,
		},
	}

	fmt.Println(pkt.String())

	s.targets[[6]byte{ip[0], ip[1], ip[2], ip[3], byte(port), byte(port >> 8)}] = wrangell{
		replicas:  replicas,
		namespace: namespace,
		name:      name,
	}

	buf := pkt.ToBuffer()
	for _, c := range s.clients {
		if c == nil {
			continue
		}
		_, err := c.Write(buf)
		if err != nil {
			log.Printf("failed to send packet to client: %v", c.RemoteAddr())
		}
	}
	return
}

func NewServer(client client.Client, address string) Server {
	return &server{
		kubeClient: client,
		targets:    map[[6]byte]wrangell{},
		address:    address,
	}
}
