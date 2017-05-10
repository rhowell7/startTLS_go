package main

// go build
// sudo ./smtp --input-file testIP.txt --output-file testOut.txt --workers 10

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
    "encoding/json"
)

type ICMPPacket struct {
    Valid       bool
    TargetIPv4  string
    ReachedIPv4 string
    Data        string
}

type Hop struct {
    Hop        int    `json:"hop,omitempty"`
    IP         string `json:"ip,omitempty"`
    HopOutput  string `json:"hop_output,omitempty"`
}

type Output struct {
    TargetIP          string `json:"target_ip,omitempty"`
    Censored          bool   `json:"censored,omitempty"`
    Hops              []Hop  `json:"hops,omitempty"`
    FirstCensoredHop  int    `json:"first_censored_hop,omitempty"`
    FirstCensoredIP   string `json:"first_censored_ip,omitempty"`
    LastUncensoredHop int    `json:"last_uncensored_hop,omitempty"`
    LastUncensoredIP  string `json:"last_uncensored_ip,omitempty"`
    TcpResponse       string `json:"tcp_response,omitempty"`
}


func main() {
    //------------------------------- Set up ---------------------------------//
    var workers int
    var output_file string
    var input_file string
    flag.IntVar(&workers, "workers", 1, "number of worker routines")
    flag.StringVar(&output_file, "output-file", "results.txt", "File location to save output")
    flag.StringVar(&input_file, "input-file", "ipAddresses.txt", "Input file should be a list of IP Addresses")
    flag.Parse()
    minTTL := 10
    maxTTL := 26
    w := new(sync.WaitGroup)
    wg := new(sync.WaitGroup)
    input_chan := make(chan string, workers*2)
    icmp_chan := make(chan ICMPPacket, workers*2)
    output_chan := make(chan Output, workers*2)

    
    //------------------- Build the queue of IP Addresses --------------------//
    // fmt.Println("Opening input_file: ", input_file)
    file, err := os.Open(input_file)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    
    // Input goroutine: put IP addresses into input_chan
    w.Add(1)
    go func() {
        defer w.Done()
        defer fmt.Println("Input goroutine has finished")
        for scanner.Scan() {
            ip := string(scanner.Text())
            input_chan <- ip
        }
        close(input_chan) // closes chan (for range() will stop when this chan closes)
    }() // Input goroutine

    
    //--------- Goroutine to print the JSON results to file or stdout --------//
    wg.Add(1)
    go func() {
        f, err := os.OpenFile(output_file, os.O_APPEND|os.O_WRONLY, 0600)
        if err != nil {
            panic(err)
        }
        defer f.Close()
        for out := range output_chan {
            b, e := json.Marshal(&out)
            if e == nil {
                // fmt.Println(string(b))
                if _, err = f.WriteString(string(b)); err != nil {
                    panic(err)
                }
            }
        }
        fmt.Println("Output goroutine has finished")
        wg.Done()
        
    }()


    //----------------------- Create the ICMP Listener -----------------------//
    w.Add(1)
    go func() {
        // defer w.Done()
        // defer fmt.Println("ICMP Listener goroutine has finished")
        icmp_conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
        icmp_duration := time.Duration(3) * time.Second
        icmp_timeout := time.Now().Add(icmp_duration)
        icmp_conn.SetReadDeadline(icmp_timeout)
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
                    // continue
                    break
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

                icmp_parsed := ICMPPacket{TargetIPv4: header.Dst.String(), ReachedIPv4: peer.(*net.IPAddr).IP.String(), Valid: true}

                if len(text) > 52 {
                    icmp_parsed.Data = text[52:]
                }
                icmp_chan <- icmp_parsed
            // default:
            //     // log.Printf("unknown ICMP message: %+v\n", rm)
            //     fmt.Println("ICMP response unknown/default/other")
            //     default_icmp_packet := ICMPPacket{Valid: false}
            //     icmp_chan <- default_icmp_packet
            } //  switch
        } // for
        w.Done()
        fmt.Println("ICMP Listener goroutine has finished")
        close(icmp_chan)
    }() // ICMP Listener


    //--------------------------- ICMP Dispatcher ----------------------------//
    in_process := make(map[string](chan ICMPPacket))
    m := new(sync.RWMutex)
    w.Add(1)
    go func() {
        // Dispatch packets to their worker chans
        defer w.Done()
        defer fmt.Println("ICMP Dispatcher goroutine has finished")
        for packet := range icmp_chan {
            m.Lock() // lock accesses to in_process map
            target := packet.TargetIPv4 // grab the target IP from the parsed ICMP 
            _, ok := in_process[target]; 
            m.Unlock()
            if ok { // if we have a worker for that target
                in_process[target] <- packet // tell the worker we got that packet
            }
        }
    }() // ICMP Dispatcher



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
            defer fmt.Println("Worker goroutine has finished")
            for ip := range input_chan {
                //------------- Get an IP Address from input_chan ------------//
                m.Lock()
                in_process[ip] = c
                m.Unlock()
                defer func() {
                    m.Lock()
                    delete(in_process, ip)
                    m.Unlock()
                }()

                target := ip+":25"
                fmt.Print("\nTarget: ", ip)

                //-------------------- Open the connection -------------------//
                timeOut := time.Duration(3) * time.Second
                conn, err := net.DialTimeout("tcp", target, timeOut)
                if err != nil {
                    fmt.Println("\ndial error:", err)
                    continue
                }
                defer conn.Close()
                
                // Wait for 220 banner
                banner, err := bufio.NewReader(conn).ReadString('\n')
                // fmt.Println(banner)

                // Are we being greylisted/blacklisted?
                bannerGood, err := regexp.MatchString("220 ", string(banner))

                if !bannerGood {
                    fmt.Println("\nThis server did not give us a good banner: ", target)
                    fmt.Println(banner)
                    continue
                }

                //---------------- Send EHLO, receive Extensions -------------//
                conn.Write([]byte("EHLO ME\r\n"))
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
                        // Extensions start with 250-, except the last one is just 250
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
                } // for (receiving extensions)


                // //----------------- Test Regular STARTTLS request ---------------------//
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


                
            //----------------- Send magic StartTLS packets ------------------//
                tcpResponseChan := make(chan []byte)
                hostTimeoutChan := make(chan bool)

                //--------------------- TCP Listener -------------------------//
                w.Add(1)
                go func() {
                    defer w.Done()
                    defer fmt.Println("TCP Listener goroutine has finished")
                    tlsResponse, err := bufReader.ReadBytes('\n')
                    if err != nil { // if there's an error
                        fmt.Println()
                        // break
                        hostTimeoutChan <- true
                    }
                    tcpResponseChan <- tlsResponse
                }() // TCP Listener

                hostDone := false
                results := Output{TargetIP: ip}

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
                    startTlsConn.Write([]byte("STARTTLS\r\n"))
                    // fmt.Println("\n\nSent STARTTLS packet with TTL: ", ttl)
                    hopDone := false


                    for !hopDone && !hostDone {
                        select {
                            case icmpPkt := <- c:
                                // got an icmp packet, add it to results
                                hopDone = true
                                hop_ip := icmpPkt.ReachedIPv4
                                hop_response := strings.Replace(icmpPkt.Data, "\n", "", -1)
                                hop_response = strings.Replace(hop_response, "\u0000", "", -1)


                                hop_result := Hop{IP: hop_ip, HopOutput: hop_response, Hop: ttl}
                                results.Hops = append(results.Hops, hop_result)

                                hop_censored, _ := regexp.MatchString("XXX", hop_response)
                                if hop_censored && results.FirstCensoredIP == "" {
                                    // fmt.Println("Found the first censored hop: ", ttl, ": ", hop_ip)
                                    results.FirstCensoredHop = ttl
                                    results.FirstCensoredIP = hop_ip
                                }

                                hop_uncensored, _ := regexp.MatchString("STARTTLS", hop_response)
                                if hop_uncensored {
                                    // fmt.Println("Found an uncensored hop: ", ttl, ": ", hop_ip)
                                    results.LastUncensoredHop = ttl
                                    results.LastUncensoredIP = hop_ip
                                }

                                fmt.Print("Hop ", ttl, ": ", hop_ip)
                                fmt.Println("\tICMP Packet: ", hop_response)

                                continue
                            case tcpBytes := <- tcpResponseChan:
                                // got a TCP packet, add it to results and break
                                hostDone = true
                                hopDone = true
                                
                                tcpResponse := string(tcpBytes)

                                hop_censored, _ := regexp.MatchString("XXX", tcpResponse)
                                if hop_censored && results.FirstCensoredIP == "" {
                                    // fmt.Println("Found the first censored hop: ", ttl, ": ", ip)
                                    results.FirstCensoredHop = ttl
                                    results.FirstCensoredIP = ip
                                    results.TcpResponse = tcpResponse
                                    results.Censored = true
                                }

                                output_chan <- results

                                fmt.Print("Hop ", ttl, ": ")
                                fmt.Print(ip)
                                fmt.Println("\tTCP Response: ", tcpResponse)
                                break
                            case <-time.After(1 * time.Second):
                                hopDone = true
                                continue
                            case <-time.After(25 * time.Second):
                                hostDone = true
                                fmt.Println("Host timed out after 25 seconds")
                                break

                        } // select
                    } // for !hopDone && !hostDone
                } // for ; ttl < maxTTL && !hostDone
            } // for ip := range input_chan
            close(c)
        }(c) // Worker goroutine
    } // for worker < workers
    w.Wait()
    close(output_chan)
    wg.Wait()
    

} // main
