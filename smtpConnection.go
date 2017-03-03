package main

import (
    "net"
    "fmt"
    "time"
    "bufio"
    "golang.org/x/net/ipv4"
    // go get -u golang.org/x/net/ipv4
)

func main() {
    // TODO: get the next IP address from a global queue
    target := "mail.rchowell.net:25"
    minTTL := 10
    maxTTL := 15

    //------------------------- Open the connection --------------------------//
    // Dial: returns a new Client connected to a server at addr
    conn, err := net.Dial("tcp", target)
    if err != nil {
        fmt.Println("dial error:", err)
        return
    }
    
    // Wait for 220 banner
    banner, err := bufio.NewReader(conn).ReadString('\n')
    fmt.Println(banner)

    //---------------------- Send EHLO, receive Extensions -------------------//
    conn.Write([]byte("EHLO ME\r\n"))
    timeoutDuration := 1 * time.Second
    bufReader := bufio.NewReader(conn)
    for {
        // Set a deadline for reading. Read operation will fail if no data
        // is received after deadline.
        conn.SetReadDeadline(time.Now().Add(timeoutDuration))

        // Read tokens delimited by newline
        extensions, err := bufReader.ReadBytes('\n')
        if err != nil {
            fmt.Println()
            break
        }

        fmt.Printf("%s", extensions)
    }

    //--------------------- Send magic StartTLS packets ----------------------//
    // Make a new ipv4 connection from the original one
    startTlsConn := ipv4.NewConn(conn)

    // Send the fudged StartTLS packets
    ttl := minTTL
    for ; ttl < maxTTL; ttl++ {
        startTlsConn.SetTTL(ttl)
        fmt.Fprintf(startTlsConn, "STARTTLS\r\n") // send packet

        // TODO: listen for reply, wait for signal from ICMP listener
        time.Sleep(100 * time.Millisecond)
    }

    //------------------------ Close the connection --------------------------//
    time.Sleep(2 * time.Second)
    defer fmt.Println("Closing Connection")
    startTlsConn.SetTTL(60)
    defer conn.Close()
    defer startTlsConn.Close()
}
