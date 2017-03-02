package main

import (
    "net"
    "fmt"
    "time"
    // "net/smtp"
    "log"
    // "crypto/tls"
    "golang.org/x/net/ipv4"
    // go get -u golang.org/x/net/ipv4
    // "github.com/google/gopacket"
    // "github.com/google/gopacket/layers"
    // "github.com/google/gopacket/examples/util"
    // go get github.com/google/gopacket
    // # Pcap dev headers might be necessary
    // sudo apt-get install libpcap-dev
    // "flag"
    "bufio"
)

func main() {
    // defer util.Run()()
    // get the next IP address
    target := "mail.rchowell.net:25"
    // localhost := "localhost"
    minTTL := 10
    maxTTL := 15

    // Dial: returns a new Client connected to an SMTP server at addr
    // smtpClient, erro := smtp.Dial(target)
    c, err := net.Dial("tcp", target)
    if err != nil {
        log.Fatal(err)
    }

    // TODO: Wait for 220 banner
    time.Sleep(1 * time.Second)

    // fmt.Printf("smtpClient:\n%+v\n", smtpClient)
    fmt.Fprintf(c, "EHLO ME\r\n\r\n")
    status, err := bufio.NewReader(c).ReadString('\n')
    fmt.Println(status)

    // TODO: Wait for SMTP Extensions
    time.Sleep(2 * time.Second)

    // Make a new ipv4 connection from the original smtp one
    startTlsConn := ipv4.NewConn(c)

    ttl := minTTL
    for ; ttl < maxTTL; ttl++ {
        startTlsConn.SetTTL(ttl)
        fmt.Fprintf(startTlsConn, "STARTTLS") // send packet

        time.Sleep(100 * time.Millisecond)
        // listen for reply:
        // status, err := bufio.NewReader(startTlsConn).ReadString('\n')
        // fmt.Println(status)
        // if err != nil {
        //     fmt.Println(err)
        // }
    }

    // Figure out our source port, seq and ack numbers
    // fmt.Printf("smtpClient.Text:\n%+v\n", smtpClient.Text)
    // fmt.Printf("smtpClient.conn:\n%+v\n", smtpClient.conn)
    // myConn := smtpClient.Text.conn
    // fmt.Printf("myConn.Text.conn:\n%+v\n", myConn.Text.conn)
    // fmt.Printf("smtpClient.Text.Reader.R:\n%+v\n", smtpClient.Text.Reader.R)
    // fmt.Printf("smtpClient.Text.Writer.W:\n%+v\n", smtpClient.Text.Writer.W)
    // fmt.Printf("smtpClient.Text.Pipeline:\n%+v\n", smtpClient.Text.Pipeline)

    // Send StartTLS (TTL-altered packets)
    // dummyTlsConfig := &tls.Config
    // smtpClient.StartTLS(dummyTlsConfig)


    // Close the connection
    time.Sleep(2 * time.Second)
    defer fmt.Println("Closing Connection")
    // defer smtpClient.Close()
    defer c.Close()
}


// func sendStartTlsPacket(ttl uint8) {
//     // -----------------------------------------------------------------------//
//     //----------------------- Create the StartTLS packet ---------------------//
//     var srcIP, dstIP net.IP
//     var srcIPstr string = "10.0.0.139" // TODO: get my source IP
//     // var srcIPstr string = "127.0.0.1" // TODO: get my source IP
//     var dstIPstr string = "41.231.120.150"
//     // var dstIPstr string = "127.0.0.1"

//     // source ip
//     srcIP = net.ParseIP(srcIPstr)
//     if srcIP == nil {
//         log.Printf("non-ip target: %q\n", srcIPstr)
//     }
//     srcIP = srcIP.To4()
//     if srcIP == nil {
//         log.Printf("non-ipv4 target: %q\n", srcIPstr)
//     }

//     // destination IP
//     dstIP = net.ParseIP(dstIPstr)
//     if dstIP == nil {
//         log.Printf("non-ip target: %q\n", dstIPstr)
//     }
//     dstIP = dstIP.To4()
//     if dstIP == nil {
//         log.Printf("non-ipv4 target: %q\n", dstIPstr)
//     }

//     // build tcp/ip packet
//     ip := layers.IPv4 {
//         SrcIP:    srcIP,
//         DstIP:    dstIP,
//         Version:  4,
//         TTL:      ttl,
//         Protocol: layers.IPProtocolTCP,
//     }
    
//     srcport := layers.TCPPort(35360) // TODO: get my source port
//     dstport := layers.TCPPort(25)
//     tcp := layers.TCP{
//         SrcPort: srcport,
//         DstPort: dstport,
//         Window:  1505,
//         Urgent:  0,
//         Seq:     11050,
//         Ack:     0,
//         ACK:     false,
//         SYN:     false,
//         FIN:     false,
//         RST:     false,
//         URG:     false,
//         ECE:     false,
//         CWR:     false,
//         NS:      false,
//         PSH:     false,
//     }

//     opts := gopacket.SerializeOptions{
//         FixLengths:       true,
//         ComputeChecksums: true,
//     }

//     tcp.SetNetworkLayerForChecksum(&ip)

//     ipHeaderBuf := gopacket.NewSerializeBuffer()
//     err := ip.SerializeTo(ipHeaderBuf, opts)
//     if err != nil {
//         panic(err)
//     }
//     ipHeader, err := ipv4.ParseHeader(ipHeaderBuf.Bytes())
//     if err != nil {
//         panic(err)
//     }

//     tcpPayloadBuf := gopacket.NewSerializeBuffer()
//     payload := gopacket.Payload([]byte("STARTTLS"))
//     err = gopacket.SerializeLayers(tcpPayloadBuf, opts, &tcp, payload)
//     if err != nil {
//         panic(err)
//     }

//     // -----------------------------------------------------------------------//
//     //----------------------- Send the StartTLS packet -----------------------//
//     time.Sleep(500 * time.Millisecond) // 0.5 Second
//     var packetConn net.PacketConn
//     var rawConn *ipv4.RawConn
//     // packetConn, err = net.ListenPacket("ip4:tcp", "41.231.120.150")
//     packetConn, err = net.ListenPacket("ip4:tcp", "0.0.0.0")
//     if err != nil {
//         panic(err)
//     }
//     rawConn, err = ipv4.NewRawConn(packetConn)
//     if err != nil {
//         panic(err)
//     }

//     err = rawConn.WriteTo(ipHeader, tcpPayloadBuf.Bytes(), nil)
//     log.Printf("packet of length %d sent!\n", (len(tcpPayloadBuf.Bytes()) + len(ipHeaderBuf.Bytes())))
// } // sendPacket