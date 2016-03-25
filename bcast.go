package main

import (
    "net"
    "strconv"
    //"bufio"
    "fmt"
    "time"
    "math"
    "encoding/json"
    "io/ioutil"

    "github.com/santegoeds/oanda"
)

type dataType struct {
    val float64
    sd float64
}
func (d dataType) toString() string {
        return (strconv.FormatFloat(float64(d.val), 'f', -1, 64) + ", " +
                strconv.FormatFloat(float64(d.sd), 'f', -1, 64))
    }

func main() {
    server, _ := net.Listen("tcp", ":3540")
    if server == nil {
        panic("Failed to open listening port")
    }

    client := connectToAPI()

    joiningClients := make(chan net.Conn)
    d := make(chan dataType, 1024);
    proc := make(chan float64)

    go acceptor(server, joiningClients);
    go handler(joiningClients, d);
    go processData(proc, d)
    historicalData(client, proc);
}

func handler(joiningClients chan net.Conn, input chan dataType) {
    conn := make([]net.Conn, 0);
    for {
        select {
            case jc := <-joiningClients:
                conn = append(conn, jc)
            case i := <-input:
                for _, elem := range conn {
                    elem.Write([]byte("" + i.toString() + "\n"))
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

var access_token string = "access_token"
var accountId oanda.Id = "accountId"

func connectToAPI() *oanda.Client {
    client, err := oanda.NewFxPracticeClient(access_token)
    if err != nil {
        panic(err)
    }

    client.SelectAccount(1956907)

    return client
}

func liveData(client *oanda.Client, proc chan float64) {
    priceServer, err := client.NewPriceServer("GBP_USD")
    if err != nil {
        panic(err)
    }

    var lastValue float64 = 0
    priceServer.HeartbeatFunc = func(hb oanda.Time) {
        proc <- lastValue;
    }
    priceServer.ConnectAndHandle(func(instrument string, tick oanda.PriceTick) {
        lastValue = tick.Bid;
    })

}

func historicalData(client *oanda.Client, proc chan float64) {
    type StreamType struct {
        Instrument string
        Granularity string
        Candles []struct {
            Time time.Time
            Bid float64
            Complete bool
        }
    }
    var data StreamType
    file, err := ioutil.ReadFile("dataD2002.json")
    if err != nil {
        panic(err.Error())
    }
    err = json.Unmarshal(file, &data)
    if err != nil {
        fmt.Println(err)
    }
    
    for {
        for i := 0; i < len(data.Candles); i++ {
            proc <- data.Candles[i].Bid
            time.Sleep(2 * time.Second)
        }
    }
}

var sampleSize, n = 50, 0
var runningMean, runningSumSq float64 = 0, 0
var queue = make([]float64, 0)

func processData(in chan float64, out chan dataType) float64 {
    for {
        lastVal := <-in
        queue = append(queue, lastVal)
        oldMean := runningMean

        if(n < sampleSize) {
            n += 1
            runningMean += (lastVal - runningMean)/float64(n)
            runningSumSq += (lastVal - runningMean)*(lastVal - oldMean)
        } else {
            oldVal := queue[0]; queue = queue[1:]
            runningMean += (lastVal - oldVal)/float64(sampleSize)

            runningSumSq = 0
            for i := 0; i < sampleSize; i++ {
                runningSumSq += (queue[i] - runningMean)*(queue[i] - runningMean)
            }
        }
        var sd float64 = 0
        if(n > 1) {
            sd = math.Sqrt(runningSumSq/float64(n-1))
        }
        out <- dataType{lastVal, sd}
        fmt.Printf("%f, %f written\n", lastVal, sd)
    }
}

