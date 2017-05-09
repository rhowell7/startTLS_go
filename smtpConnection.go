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
    "os"
    // "reflect"
    "sync"
    "flag"
    "strings"
)

type ICMPPacket struct {
    Valid       bool
    // TargetIPv4  net.IP
    // ReachedIPv4 net.Addr
    TargetIPv4  string
    ReachedIPv4 string
    Data        string
    // TTL         int
}

func main() {
    //---------------------------- Set up ------------------------------//
    // workers := 1 // number of threads, should be a flag // ./smtp --workers 10
    var workers int
    flag.IntVar(&workers, "workers", 1, "number of worker routines")
    minTTL := 10
    maxTTL := 26
    w := new(sync.WaitGroup)
    input_chan := make(chan string, workers*2)
    icmp_chan := make(chan ICMPPacket, workers*2)

    
    //------------------- Build the queue of IP Addresses --------------------//
    file, err := os.Open("ipAddresses.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close() // closes file

    scanner := bufio.NewScanner(file)
    
    // Input goroutine: put IP addresses into input_chan
    w.Add(1)
    go func() {
        defer w.Done()
        for scanner.Scan() {
            // fmt.Println("57: Putting IP into input_chan: ", scanner.Text()) // fmt.Println(scanner.Text())
            ip := string(scanner.Text())
            // fmt.Println(ip)
            // fmt.Println(reflect.TypeOf(ip)) // string
            input_chan <- ip
            // fmt.Println("Put one IP into input_chan")
        }
        close(input_chan) // closes chan (for range() will stop when this chan closes)
    
        // fmt.Println("Done reading in IP Addresses\n")

        // if err := scanner.Err(); err != nil {
        //     log.Fatal(err)
        // }
    }() // Input goroutine

    // fmt.Println("scanner read in ip addresses:")
    // for i := 0; i < len(input_chan); i++ {
    //     fmt.Println(<-input_chan)
    // }


    //----------------------- Create the ICMP Listener -----------------------//
    w.Add(1)
    go func() {
        defer w.Done()
        icmp_conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
        if err != nil {
            log.Fatal(err)
        }
        x := 8
        for {
            rb := make([]byte, 10000)
            n, peer, err := icmp_conn.ReadFrom(rb)
            if err != nil {
                if err, ok := err.(net.Error); ok && err.Timeout() {
                    fmt.Printf("%v\t*\n", 3)
                    continue
                }
                log.Fatal(err)
            }

            icmp_packet := rb[:n]

            rm, err := icmp.ParseMessage(1, icmp_packet)

            if err != nil {
                fmt.Println("fatal error parsing ICMP message")
                log.Fatal(err)
            }
            switch rm.Type {
            case ipv4.ICMPTypeTimeExceeded:
                // fmt.Printf("peer: %v\n", peer)
                body := rm.Body.(*icmp.TimeExceeded)
                text := string(body.Data)

                // fmt.Println("Parsing rb[x:], where x = ", x)
                header, err := ipv4.ParseHeader(rb[x:])
                if err != nil {
                    fmt.Println("fatal error parsing ICMP message: icmp_header")
                    log.Fatal(err)
                }
                // fmt.Println("TARGET: icmp_packet.Dst: ", header.Dst)
                // fmt.Println("REACHED: peer: ", peer)
                // fmt.Print("DATA: ")


                // icmp_parsed := ICMPPacket{TargetIPv4: header.Dst.To4(), ReachedIPv4: peer.(*net.TCPAddr).IP.To4(), Valid: true}
                // icmp_parsed := ICMPPacket{TargetIPv4: header.Dst.String(), ReachedIPv4: peer.(*net.TCPAddr).IP.String(), Valid: true}
                icmp_parsed := ICMPPacket{TargetIPv4: header.Dst.String(), ReachedIPv4: peer.(*net.IPAddr).IP.String(), Valid: true}


                // icmp_extensions := body.Extensions
                if len(text) > 52 {
                    // fmt.Print("Hop", ttl, ": ", peer)
                    // fmt.Println(peer)
                    // fmt.Println("\tICMP response: ", string(text[52:]))
                    // fmt.Println("ICMP extensions: ", icmp_extensions)
                    icmp_parsed.Data = text[52:]
                } else {
                    // fmt.Println("Peer did not include a response :(")
                }
                icmp_chan <- icmp_parsed
            // case ipv4.ICMPTypeEchoReply:
            //     // icmp_chan <- "echo"
            //     names, _ := net.LookupAddr(peer.String())
            //     fmt.Println("\t%v %+v \n\t%+v", peer, names, rm)
            //     fmt.Println("195: got an echo reply")
            //     // return // kill this goroutine, & defer'd still runs
            // default:
            //     // log.Printf("unknown ICMP message: %+v\n", rm)
            //     fmt.Println("ICMP response unknown/default/other")
            //     default_icmp_packet := ICMPPacket{Valid: false}
            //     icmp_chan <- default_icmp_packet
            } //  switch
        } // for
        close(icmp_chan)
    }() // ICMP Listener


    //--------------------------- ICMP Dispatcher ----------------------------//
    in_process := make(map[string](chan ICMPPacket))
    m := new(sync.RWMutex)
    w.Add(1)
    go func() {
        // Dispatch packets to scanners
        defer w.Done()
        for packet := range icmp_chan {
            // TODO: This lock was causing problems... Do we need it?????? ???????????
            // m.Lock() // lock accesses to in_process map
            // defer m.Unlock()
            target := packet.TargetIPv4 // grab the target IP from the parsed ICMP 
            if _, ok := in_process[target]; ok { // if we have a worker for that target
                in_process[target] <- packet // tell the worker we got that packet
            }
        }
    }() // ICMP Dispatcher





    // -------TODO: get the next IP address from a global queue ---------------------//
    // target := "mail.rchowell.net:25"
    // target := "128.32.80.167:25"
    // target := "173.194.69.26:25"
    // target := "128.32.78.35:25"
    // target := "128.248.41.50:25"
    // target := "66.196.118.35:25"
    // target := "65.55.33.135:25"
    // target := "128.32.78.14:25"
    // target := "131.193.46.40:25" // dial timeout




    //------------------------------ Worker Goroutine ------------------------//
    worker_channels := make([]chan ICMPPacket, workers)
    for idx := range worker_channels {
        worker_channels[idx] = make(chan ICMPPacket, 1)
    }

    for worker := 0; worker < workers; worker++ {
        w.Add(1)
        c := worker_channels[worker]
        go func(c chan ICMPPacket) {
            defer w.Done()
            for ip := range input_chan {
                // fmt.Println("201: Getting new IP Address")
                //------------- Get an IP Address from input_chan ------------//
                m.Lock()
                // fmt.Println("204")
                in_process[ip] = c
                // fmt.Println("206")
                m.Unlock()
                // fmt.Println("208")
                defer func() {
                    // fmt.Println("210")
                    m.Lock()
                    // fmt.Println("212")
                    delete(in_process, ip)
                    // fmt.Println("214")
                    m.Unlock()
                    // fmt.Println("216")
                }()

                target := ip+":25"
                fmt.Print("\nTarget: ", ip)

                //------------------------ Open the connection -----------------------//
                // fmt.Println("\n\nTarget: ", target)
                timeOut := time.Duration(3) * time.Second
                // Dial: returns a new Client connected to a server at addr
                // conn, err := net.Dial("tcp", target)
                conn, err := net.DialTimeout("tcp", target, timeOut)
                if err != nil {
                    fmt.Println("\ndial error:", err)
                    // conn.Close()
                    // break
                    // return
                    continue
                }
                // defer fmt.Println("Closing Connection, test")
                defer conn.Close()
                
                // Wait for 220 banner
                banner, err := bufio.NewReader(conn).ReadString('\n')
                // fmt.Println(banner)

                // Are we being greylisted/blacklisted?
                bannerGood, err := regexp.MatchString("220 ", string(banner))

                if !bannerGood {
                    fmt.Println("\nThis server did not give us a good banner: ", target)
                    // return
                    continue
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

                    // fmt.Printf("%s", extensions)

                    matched, err := regexp.MatchString("250 ", string(extensions))
                    if matched {
                        // fmt.Println("Found the last extension\n")
                        fmt.Println()
                        break
                    }

                    // 500 5.5.1 Command unrecognized: "XXXX ME"
                    ehloError, err := regexp.MatchString("500 ", string(extensions))
                    if ehloError {
                        // fmt.Println("got a 500 error: ", string(extensions))
                        // fmt.Println("Sending 'HELO' instead")
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


                
                //--------------------- Send magic StartTLS packets ----------------------//
                tcpResponseChan := make(chan []byte)
                hostTimeoutChan := make(chan bool)

                //--------------------- TCP Listener -------------------------//
                // w.Add(1)
                // go func() { // TCP Listener
                //     defer w.Done()
                //     timeoutDuration := 5 * time.Second
                //     bufReader := bufio.NewReader(conn)
                //     conn.SetReadDeadline(time.Now().Add(timeoutDuration))
                //     res, err := bufReader.ReadBytes('\n')
                //     // res, err := ReadNormalStartTLSAnswerWithMaxTimeout(conn)
                //     if err != nil {
                //         fmt.Println(err)
                //         hostTimeoutChan <- true
                //     }
                //     tcpResponseChan <- res
                // }() // TCP Listener

                //--- Listen for a TLS response ---//
                w.Add(1)
                go func() {
                    defer w.Done()
                    tlsResponse, err := bufReader.ReadBytes('\n')
                    if err != nil { // if there's an error
                        fmt.Println()
                        // break
                        hostTimeoutChan <- true
                    }
                    // fmt.Print("Got TCP response:\t")
                    // fmt.Println(string(tlsResponse))
                    tcpResponseChan <- tlsResponse
                }()



                hostDone := false
                // ttlCount := 0



                // Make a new ipv4 connection from the original one
                startTlsConn := ipv4.NewConn(conn)
                if err != nil {
                    fmt.Println("Forking conn error:", err)
                    return
                }
                defer startTlsConn.Close()

                // Send the fudged StartTLS packets
                ttl := minTTL
                for ; ttl < maxTTL && !hostDone; ttl++ {
                    startTlsConn.SetTTL(ttl)
                    // fmt.Fprintf(startTlsConn, "STARTTLS\r\n") // send packet
                    startTlsConn.Write([]byte("STARTTLS\r\n"))
                    // fmt.Println("\n\nSent STARTTLS packet with TTL: ", ttl)
                    // fmt.Print("Hop ", ttl, ": ")
                    hopDone := false


                    for !hopDone && !hostDone {
                        select {
                            case icmpPkt := <- c:
                            // case <- c:
                                // got an icmp packet
                                hopDone = true
                                fmt.Print("Hop ", ttl, ": ", icmpPkt.ReachedIPv4)
                                // fmt.Println("\tICMP Packet: ", icmpPkt.Data)
                                fmt.Println("\tICMP Packet: ", strings.Replace(icmpPkt.Data, "\n", "", -1))

                                // check for censorship in echo'd response
                                // fmt.Println("case icmpPkt: icmpPkt.Data: ", icmpPkt.Data)
                                continue
                            case tcpBytes := <- tcpResponseChan:
                                hostDone = true
                                hopDone = true
                                // fmt.Println("case tcpBytes: ", tcpBytes)
                                tcpResponse := string(tcpBytes)
                                fmt.Print("Hop ", ttl, ": ")
                                header, err := ipv4.ParseHeader(tcpBytes)
                                if err != nil {
                                    fmt.Println("Error parsing tcpResponse")
                                    break
                                }
                                // fmt.Println("Got TCP response from: ")
                                fmt.Print(header.Src)
                                fmt.Println("\tTCP Response: ", tcpResponse)
                                break
                                // ttlCount = ttl
                            case <-time.After(1 * time.Second):
                                hopDone = true
                                continue
                            // case <- hostTimeoutChan:
                            case <-time.After(25 * time.Second):
                                hostDone = true
                                fmt.Println("Host timed out after 25 seconds")
                                break

                        } // select
                    }


                    // TODO: listen for reply, wait for signal from ICMP listener
                    // time.Sleep(100 * time.Millisecond)
                    // fmt.Println("224: waiting for icmp_chan")
                    // starttlsResponse := <-icmp_chan
                    // fmt.Println("226: got icmp_chan")
                    // if starttlsResponse == "ttl_expired" {
                    // if starttlsResponse.Valid {
                    //     fmt.Println("TTL expired, loop again")
                    // } else {
                    //     fmt.Println("listen for the normal STARTTLS response now")
                    //     tlsResponse, err := bufio.NewReader(conn).ReadBytes('\n')
                    //     if err != nil {
                    //         fmt.Println("Error listening for StartTLS response: ", err)
                    //     }
                    //     fmt.Println(string(tlsResponse))

                    //     starttlsGood, err := regexp.MatchString("220 ", string(tlsResponse))
                    //     starttlsBad, err := regexp.MatchString("500 ", string(tlsResponse))

                    //     if starttlsGood {
                    //         fmt.Println("This server is good to start TLS: ", target)
                    //     } else if starttlsBad {
                    //         fmt.Println("This server is unable to start TLS: ", target)
                    //     } else {
                    //         fmt.Println("Response did not contain 220 or 500, trying again")
                    //         // conn.Write([]byte("STARTTLS\n"))
                    //         // tlsResponse, err = bufio.NewReader(conn).ReadBytes('\n')
                    //         // fmt.Println(string(tlsResponse))
                    //     }
                    // }
                }
            } // for ip := range input_chan
            close(c)
        }(c) // Worker goroutine
    } // for worker < workers
    w.Wait()

    //------------------------ Close the connection --------------------------//
    // time.Sleep(1 * time.Second)
    // defer fmt.Println("Closing Connection, test")
    // startTlsConn.SetTTL(60)
    // defer conn.Close() // done at the top
    // defer startTlsConn.Close()
}
