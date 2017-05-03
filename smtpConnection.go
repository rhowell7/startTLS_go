package main

// go build
// ./smtp

import (
    "net"
    "regexp"
    "fmt"
    "time"
    "bufio"
    "golang.org/x/net/ipv4"
    // go get -u golang.org/x/net/ipv4
    "golang.org/x/net/icmp"
    "log"
    // "os"
    // "reflect"
)

func main() {
    //---------------------------- Set up ------------------------------//
    // workers := 10 // number of threads, should be a flag


    
    //------------------- Build the queue of IP Addresses --------------------//
    file, err := os.Open("ipAddresses.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close() // closes file

    ipAddresses := make(chan string)
    scanner := bufio.NewScanner(file)
    
    // in a goroutine
    // for scanner.Scan() {
    //     fmt.Println(scanner.Text())
    //     tmp := string(scanner.Text())
    //     fmt.Println(tmp)
    //     fmt.Println(reflect.TypeOf(tmp))
    //     ipAddresses <- tmp
    //     fmt.Println("read one IP")
    // }
    // close(ipAddresses) // closes chan (for range() will stop when this chan closes)
    
    fmt.Println("Done reading in IP Addresses\n")

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    fmt.Println("scanner read in ip addresses:\n")
    for i := 0; i < len(ipAddresses); i++ {
        fmt.Println(<-ipAddresses)
    }


    // -------TODO: get the next IP address from a global queue ---------------------//
    target := "mail.rchowell.net:25"
    // target := "128.32.80.167:25"
    // target := "173.194.69.26:25"
    // target := "128.32.78.35:25"
    // target := "128.248.41.50:25"
    // target := "66.196.118.35:25"
    // target := "65.55.33.135:25"
    // target := "128.32.78.14:25"
    // target := "131.193.46.40:25" // dial timeout

    minTTL := 10
    maxTTL := 26

    //------------------------- Open the connection --------------------------//
    fmt.Println("Attempting connection to: ", target)
    timeOut := time.Duration(5) * time.Second
    // Dial: returns a new Client connected to a server at addr
    // conn, err := net.Dial("tcp", target)
    conn, err := net.DialTimeout("tcp", target, timeOut)
    if err != nil {
        fmt.Println("dial error:", err)
        return
    }
    defer fmt.Println("Closing Connection, test")
    defer conn.Close()
    
    // Wait for 220 banner
    banner, err := bufio.NewReader(conn).ReadString('\n')
    fmt.Println(banner)

    // Are we being greylisted/blacklisted?
    bannerGood, err := regexp.MatchString("220 ", string(banner))

    if !bannerGood {
        fmt.Println("This server did not give us a good banner: ", target)
        return
    }

    //---------------------- Send EHLO, receive Extensions -------------------//
    // time.Sleep(100 * time.Millisecond)
    conn.Write([]byte("EHLO ME\r\n"))
    // while "250-*", reading lines. if "250 *", break
    bufReader := bufio.NewReader(conn)
    for {
        // Read tokens delimited by newline
        extensions, err := bufReader.ReadBytes('\n')
        if err != nil { // if there's an error
            fmt.Println()
            break
        }

        fmt.Printf("%s", extensions)

        matched, err := regexp.MatchString("250 ", string(extensions))
        if matched {
            // fmt.Println("Found the last extension\n")
            fmt.Println()
            break
        }

        // 500 5.5.1 Command unrecognized: "XXXX ME"
        ehloError, err := regexp.MatchString("500 ", string(extensions))
        if ehloError {
            fmt.Println("got a 500 error: ", string(extensions))
            fmt.Println("Sending 'HELO' instead")
            conn.Write([]byte("HELO ME\r\n"))
        }
    } // for


    // //----------------- Test Tegular STARTTLS request ---------------------//
    // fmt.Println("Sending STARTTLS")
    // conn.Write([]byte("STARTTLS\r\n"))
    // tlsResponse, err := bufio.NewReader(conn).ReadBytes('\n')
    // fmt.Println(string(tlsResponse))
    // // fmt.Println(err)

    // starttlsGood, err := regexp.MatchString("220 ", string(tlsResponse))
    // starttlsBad, err := regexp.MatchString("500 ", string(tlsResponse))

    // if starttlsGood {
    //     fmt.Println("This server is good to start TLS: ", target)
    // } else if starttlsBad {
    //     fmt.Println("This server is unable to start TLS: ", target)
    // } else {
    //     fmt.Println("Response did not contain 220 or 500, trying again")
    //     conn.Write([]byte("STARTTLS\n"))
    //     tlsResponse, err = bufio.NewReader(conn).ReadBytes('\n')
    //     fmt.Println(string(tlsResponse))
    // }


    //----------------------- Create the ICMP Listener -----------------------//
    icmp_chan := make(chan string)
    go func() {
        icmp_conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
        if err != nil {
            log.Fatal(err)
        }
        for {
            fmt.Println("160")
            rb := make([]byte, 15000)
            fmt.Println("161: in ICMP listener")
            n, peer, err := icmp_conn.ReadFrom(rb)
            fmt.Println("164")
            if err != nil {
                if err, ok := err.(net.Error); ok && err.Timeout() {
                    fmt.Printf("%v\t*\n", 3)
                    continue
                }
                fmt.Println("170")
                log.Fatal(err)
            }
            fmt.Println("173")
            rm, err := icmp.ParseMessage(1, rb[:n])
            fmt.Println("175")
            if err != nil {
                fmt.Println("fatal error parsing ICMP message")
                log.Fatal(err)
            }
            fmt.Println("180")
            switch rm.Type {
            case ipv4.ICMPTypeTimeExceeded:
                fmt.Printf("peer: %v\n", peer)
                body := rm.Body.(*icmp.TimeExceeded)
                text := string(body.Data)
                if len(text) > 52 {
                    fmt.Println("ICMP response: ", string(text[52:]))
                } else {
                    fmt.Println("Peer did not include a response :(")
                }
                icmp_chan <- "ttl_expired"
            case ipv4.ICMPTypeEchoReply:
                icmp_chan <- "echo"
                names, _ := net.LookupAddr(peer.String())
                fmt.Println("\t%v %+v \n\t%+v", peer, names, rm)
                fmt.Println("195: got an echo reply")
                // return // kill this goroutine, & defer'd still runs
            default:
                // log.Printf("unknown ICMP message: %+v\n", rm)
                fmt.Println("ICMP response unknown/default/other")
                icmp_chan <- "unknown"
            }
        }
    }()

    //--------------------- Send magic StartTLS packets ----------------------//
    // Make a new ipv4 connection from the original one
    startTlsConn := ipv4.NewConn(conn)
    if err != nil {
        fmt.Println("Forking conn error:", err)
        return
    }
    defer startTlsConn.Close()

    // Send the fudged StartTLS packets
    ttl := minTTL
    for ; ttl < maxTTL; ttl++ {
        startTlsConn.SetTTL(ttl)
        // fmt.Fprintf(startTlsConn, "STARTTLS\r\n") // send packet
        startTlsConn.Write([]byte("STARTTLS\r\n"))
        fmt.Println("\nSent STARTTLS packet with TTL: ", ttl)

        // TODO: listen for reply, wait for signal from ICMP listener
        // time.Sleep(100 * time.Millisecond)
        fmt.Println("224: waiting for icmp_chan")
        starttlsResponse := <-icmp_chan
        fmt.Println("226: got icmp_chan")
        if starttlsResponse == "ttl_expired" {
            fmt.Println("TTL expired, loop again")
        } else if starttlsResponse == "unknown" {
            fmt.Println("unknown response, loop again")
        } else if starttlsResponse == "echo" {
            fmt.Println("echo response, loop again")
        }else {
            fmt.Println("listen for the normal STARTTLS response now")
            tlsResponse, err := bufio.NewReader(conn).ReadBytes('\n')
            if err != nil {
                fmt.Println("Error listening for StartTLS response: ", err)
            }
            fmt.Println(string(tlsResponse))

            starttlsGood, err := regexp.MatchString("220 ", string(tlsResponse))
            starttlsBad, err := regexp.MatchString("500 ", string(tlsResponse))

            if starttlsGood {
                fmt.Println("This server is good to start TLS: ", target)
            } else if starttlsBad {
                fmt.Println("This server is unable to start TLS: ", target)
            } else {
                fmt.Println("Response did not contain 220 or 500, trying again")
                // conn.Write([]byte("STARTTLS\n"))
                // tlsResponse, err = bufio.NewReader(conn).ReadBytes('\n')
                // fmt.Println(string(tlsResponse))
            }
        }

    }

    //------------------------ Close the connection --------------------------//
    // time.Sleep(1 * time.Second)
    // defer fmt.Println("Closing Connection, test")
    // startTlsConn.SetTTL(60)
    // defer conn.Close() // done at the top
    // defer startTlsConn.Close()
}
