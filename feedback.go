package main

import (
    "net"
    "bufio"
    "fmt"
    "strings"
    "strconv"

    "golang.org/x/net/websocket"
)

/* Thread to handle feedback from an individual client */
func feedbackListener(client net.Conn, fba chan string){
    b := bufio.NewReader(client)
    for {
        line, err := b.ReadBytes('\n')
        if err != nil {
            fmt.Printf("Dropping %v\n", client.RemoteAddr())
            break
        }
        sline := strings.TrimSuffix(string(line[:]),"\n")
        fmt.Printf("%s received from %v\n", sline, client.RemoteAddr())
        fba <- sline
    }
}

/* Thread to accumulate all feedback from clients into a map each tick */
func feedbackAccumulator(fba chan string, tick chan float64, 
        output chan map[string]string){
    var m map[string]string
    for {
        select {
            case v := <-fba:
                i, _ := strconv.Atoi(m[v])
                m[v] = strconv.Itoa(i + 1)
            case tick := <-tick:
                output <- m
                m = make(map[string]string)
                m["tick"] = strconv.FormatFloat(tick, 'f', -1, 64)
        }
    }
}

/* Receives the map of feedback each tick */
func feedbackOutput(data chan map[string]string, outputWeb chan map[string]string){
    for {
        m := <-data;
        for k := range m {
            if(k != "tick") { fmt.Printf("%s had %d feedback hits\n", k, m[k]); }
        }

        select {
            case outputWeb<-m:
            default:
        }
    }
}

func socketHandler(data chan map[string]string) websocket.Handler {
  return func(ws *websocket.Conn) {
    for { 
        m := <-data;
        var res = "Tick " + m["tick"] +": "
      
        if(m["1"] == "" && m["-1"] == "") {
            res = res + "no buys/sells"
        } else {
            if(m["1"] != "") { res = res + m["1"] + " buys" }
            if(m["1"] != "" && m["-1"] != "") { res = res + " and " }
            if(m["-1"] != "") { res = res + m["-1"] + " sells" }
        }
        _, err := ws.Write([]byte(res))
        if err != nil {
            break
        }
    }
  }
}
