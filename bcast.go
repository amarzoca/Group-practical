package main

import (
    "net"
    "strconv"
    //"bufio"
    "fmt"
    //"time"
    //"math/rand"

    "github.com/santegoeds/oanda"
)

type dataType float64

func main() {
    server, _ := net.Listen("tcp", ":3540")
    if server == nil {
        panic("Failed to open listening port")
    }

    client := connectToAPI()

    joiningClients := make(chan net.Conn)
    d := make(chan dataType, 1024);

    go acceptor(server, joiningClients);
    go handler(joiningClients, d)
    data(d, client);
}

func handler(joiningClients chan net.Conn, input chan dataType) {
    conn := make([]net.Conn, 0);
    for {
        select {
            case jc := <-joiningClients:
                conn = append(conn, jc)
            case i := <-input:
                for _, elem := range conn {
                    elem.Write([]byte("" + strconv.FormatFloat(float64(i), 'f', -1, 64) + "\n"))
                }
        }
    }
}

func acceptor(listener net.Listener, output chan net.Conn) {
    for {
        client, _ := listener.Accept()
        if client == nil {
            fmt.Printf("couldn't accept")
            continue
        }
        fmt.Printf("Accepted: %v <-> %v\n", client.LocalAddr(), client.RemoteAddr())
        output <- client
    }
}

//var access_token string = "634396fa2010f81d4a362e6edc6269e1-7b55b84dac91e4da6373310aae581b87"
var access_token string = "919c4660dd9eede2373d1d00648559c5-5ff4354696aef8bdd02fac15256ebb56"
var accountId oanda.Id = 1956907

func connectToAPI() *oanda.Client {
    client, err := oanda.NewFxPracticeClient(access_token)
    if err != nil {
        panic(err)
    }

    client.SelectAccount(1956907)

    return client
}

func data(out chan dataType, client *oanda.Client) {
    // Create and run a price server.
    priceServer, err := client.NewPriceServer("GBP_USD")
    if err != nil {

        panic(err)
    }
    lastValue := float64(0)
    priceServer.HeartbeatFunc = func(hb oanda.Time) {
        //fmt.Printf("Heartbeat: %s \n", hb)
        out <- dataType(lastValue);
        fmt.Printf("%f written\n", lastValue)
    }
    priceServer.ConnectAndHandle(func(instrument string, tick oanda.PriceTick) {
        lastValue = tick.Bid;
        fmt.Printf("%f stored\n", lastValue)
    })
}