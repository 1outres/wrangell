package gateway

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"syscall"
	"time"

	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
	"github.com/1outres/wrangell/pkg/wrangellpkt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/routing"
	corev1 "k8s.io/api/core/v1"
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
			binary.BigEndian.PutUint32(ip, pkt.ReqPacket.DstIp)

			w, ok := s.targets[[6]byte{ip[0], ip[1], ip[2], ip[3], byte(pkt.ReqPacket.DstPort), byte(pkt.ReqPacket.DstPort >> 8)}]

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

					failed := false
					var pod corev1.Pod
					for {
						var pods corev1.PodList
						err := s.kubeClient.List(ctx, &pods, client.InNamespace(w.namespace), client.MatchingLabels{"wrangell.loutres.me/name": wrangell.Name})
						if err != nil {
							log.Printf("Failed to list Pods: %s in %s", w.name, w.namespace)
							failed = true
							break
						}

						if len(pods.Items) == 0 {
							log.Printf("Waiting for the container to be created: %s in %s", w.name, w.namespace)

							time.Sleep(50 * time.Millisecond)
							continue
						} else if len(pods.Items) > 1 {
							log.Printf("Too many Pods found: %s in %s", w.name, w.namespace)
							failed = true
							break
						}

						pod = pods.Items[0]

						if pod.Status.Phase != corev1.PodRunning {
							log.Printf("Waiting for the container to be running: %s in %s", w.name, w.namespace)

							time.Sleep(50 * time.Millisecond)
							continue
						}

						break
					}

					if failed {
						return
					}

					dstIp := net.ParseIP(pod.Status.PodIP)
					dstPort := wrangell.Spec.Port
					var srcPort uint16 = 12345

					router, err := routing.New()
					if err != nil {
						log.Fatal("routing error:", err)
					}
					iface, _, srcIp, err := router.Route(dstIp)

					handle, err := pcap.OpenLive(iface.Name, 1600, true, pcap.BlockForever)
					if err != nil {
						log.Fatalf("pcap open error: %v", err)
						return
					}
					defer handle.Close()

					dstHwaddr, err := s.getHwAddr(handle, iface, dstIp, srcIp)
					if err != nil {
						log.Fatalf("failed to get hardware address: %v", err)
						return
					}
					srcHwaddr := iface.HardwareAddr

					synStop := make(chan struct{})

					go func() {
						ticker := time.NewTicker(50 * time.Millisecond)
						for {
							select {
							case <-ticker.C:
								res := s.sendSYN(handle, srcHwaddr, dstHwaddr, srcIp, dstIp, uint16(srcPort), uint16(dstPort), 0)
								if !res {
									return
								}
								log.Printf("Waiting for ACK: %s in %s", w.name, w.namespace)
							case <-synStop:
								return
							}
						}
					}()

					timeout := 30 * time.Second
					timeoutChan := time.After(timeout)

					packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

				LOOP:
					for {
						select {
						case packet := <-packetSource.Packets():
							ipLayer := packet.Layer(layers.LayerTypeIPv4)
							tcpLayer := packet.Layer(layers.LayerTypeTCP)
							if ipLayer == nil || tcpLayer == nil {
								continue
							}
							synackip, ok := ipLayer.(*layers.IPv4)
							if !ok {
								continue
							}
							synack, ok := tcpLayer.(*layers.TCP)
							if !ok {
								continue
							}
							if tcpLayer != nil {
								if synackip.SrcIP.Equal(dstIp) && synack.SYN {
									log.Printf("ACK received: %s in %s", w.name, w.namespace)
									close(synStop)
									break LOOP
								}
							}
						case <-timeoutChan:
							log.Printf("ACK timeout: %s in %s", w.name, w.namespace)
							close(synStop)
							break LOOP
						}
					}

					log.Printf("Sending ACK using the initial sequence number: %s in %s", w.name, w.namespace)
					srcip := make(net.IP, 4)
					binary.BigEndian.PutUint32(srcip, pkt.ReqPacket.SrcIp)
					_ = s.sendSYN(handle, srcHwaddr, dstHwaddr, srcip, dstIp, pkt.ReqPacket.SrcPort, uint16(dstPort), pkt.ReqPacket.Seq)
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

func (s *server) getHwAddr(handle *pcap.Handle, iface *net.Interface, dst, src net.IP) (net.HardwareAddr, error) {
	start := time.Now()
	// Prepare the layers to send for an ARP request.
	eth := layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(iface.HardwareAddr),
		SourceProtAddress: []byte(src.To4()),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte(dst.To4()),
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	if err := gopacket.SerializeLayers(buf, opts, &eth, &arp); err != nil {
		return nil, err
	}
	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		return nil, err
	}
	for {
		if time.Since(start) > time.Second*3 {
			return nil, errors.New("timeout getting ARP reply")
		}
		data, _, err := handle.ReadPacketData()
		if err == pcap.NextErrorTimeoutExpired {
			continue
		} else if err != nil {
			return nil, err
		}
		packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.NoCopy)
		if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
			arp := arpLayer.(*layers.ARP)
			if net.IP(arp.SourceProtAddress).Equal(net.IP(dst)) {
				return net.HardwareAddr(arp.SourceHwAddress), nil
			}
		}
	}
}

func (s *server) sendSYN(handle *pcap.Handle, srcHwaddr, dstHwaddr net.HardwareAddr, srcIp, dstIp net.IP, srcPort, dstPort uint16, seq uint32) bool {
	eth := &layers.Ethernet{
		BaseLayer:    layers.BaseLayer{},
		SrcMAC:       srcHwaddr,
		DstMAC:       dstHwaddr,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		Flags:    layers.IPv4DontFragment,
		TTL:      64,
		SrcIP:    srcIp,
		DstIP:    dstIp,
		Protocol: layers.IPProtocolTCP,
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		SYN:     true,
		Seq:     seq,
	}
	tcp.SetNetworkLayerForChecksum(ip)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, tcp); err != nil {
		log.Printf("failed to serialize layers: %v", err)
		return false
	}

	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		log.Printf("failed to send SYN packet: %v", err)
		return false
	}

	return true
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
