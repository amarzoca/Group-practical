package main

import (
    "net"
    "strconv"
    "fmt"
    "time"
    "math"
    "encoding/json"
    "io/ioutil"

    "github.com/santegoeds/oanda"
)

/* API login info */
var access_token string = ""
var accountId oanda.Id = 0

/* The data to be sent out each tick */
type dataType struct {
    val float64 //instrument value
    sd float64 //standard deviation
}

/* Format the data as a string for sending */
func (d dataType) toString() string {
    return (strconv.FormatFloat(float64(d.val), 'f', -1, 64) + ", " +
            strconv.FormatFloat(float64(d.sd), 'f', -1, 64))
}

/* Entry point, start threads */
func main() {
    server, _ := net.Listen("tcp", ":3540")
    if server == nil {
        panic("Failed to open listening port")
    }

    joiningClients := make(chan net.Conn);
    d := make(chan dataType, 1024);
    proc := make(chan float64);
    fba := make(chan string);
    fbo := make(chan map[string]int);
    tick := make(chan int);

    go acceptor(server, joiningClients);
    go handler(joiningClients, d, fba);
    go processData(proc, d, tick)
    go historicalData(proc);
    go feedbackAccumulator(fba, tick, fbo);
    feedbackOutput(fbo);
}

/* Thread for accepting new clients. Listens for clients, accepts their connections and writes
them into a channel to be handled */
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

/* Thread for handling data transfer. Writes data from input to all clients, and accepts new clients
on joiningClients */
//TODO: Clients are not removed from the list when they leave
func handler(joiningClients chan net.Conn, input chan dataType, fba chan string) {
    conn := make([]net.Conn, 0);
    for {
        select {
            case jc := <-joiningClients:
                conn = append(conn, jc)
                go feedbackListener(jc, fba)
            case i := <-input:
                for _, elem := range conn {
                    elem.Write([]byte("" + i.toString() + "\n"))
                }
        }
    }
}

/* Make a connection to the finance api */
func connectToAPI() *oanda.Client {
    client, err := oanda.NewFxPracticeClient(access_token)
    if err != nil {
        panic(err)
    }

    client.SelectAccount(accountId)

    return client
}

/* Thread that writes live data to the processing queue channel */
func liveData(proc chan float64) {
    client := connectToAPI()

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

/* Thread that writes historical data to the processing queue channel */
func historicalData(proc chan float64) {
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


/* Thread that processes the data and writes it to the input channel for broadcasting */
var sampleSize, n = 50, 0
var runningMean, runningSumSq float64 = 0, 0
var queue = make([]float64, 0)

func processData(in chan float64, out chan dataType, tick chan int) float64 {
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
        tick <- 0
        fmt.Printf("%f, %f written\n", lastVal, sd)
    }
}

