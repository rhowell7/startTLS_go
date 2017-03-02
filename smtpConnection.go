package main

import (
    "net"
    "fmt"
    "net/smtp"
    "log"
    "golang.org/x/net/ipv4"
    // REQUIRES: go get -u golang.org/x/net/ipv4
    // "github.com/google/gopacket"
    // "github.com/google/gopacket/layers"
    // # Get the gopacket package from GitHub
    // go get github.com/google/gopacket
    // # Pcap dev headers might be necessary
    // sudo apt-get install libpcap-dev
)

func main() {
    // get the next IP address
    target := "mail.rchowell.net:25"
    localhost := "localhost"
    maxTTL := 30

    // Dial: returns a new Client connected to an SMTP server at addr
    smtpClient, err := smtp.Dial(target)
    if err != nil {
        log.Fatal(err)
    }

    // Hello: send EHLO/HELO
    smtpClient.Hello(localhost)

    // Send TTL-altered packets
    magicPacket()
    listener, err := net.Listen("tcp", "0.0.0.0:0")
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    newConnection, err := listener.Accept()
    startTlsConnection := ipv4.NewConn(newConnection)
    for i := 4; i < maxTTL; i++ {
        startTlsConnection.SetTTL(i)
        startTlsConnection.Write([]byte("STARTTLS"))
    }


    // Close: close the connection
    defer fmt.Println("Closing Connection")
    defer smtpClient.Close()
}


func magicPacket() {
    fmt.Println("Sending magic packets")
}