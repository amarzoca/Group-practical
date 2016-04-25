package main

import (
    "net"
    "bufio"
    "fmt"
    "strings"
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
func feedbackAccumulator(fba chan string, tick chan int, output chan map[string]int){
    var m map[string]int
    for {
        select {
            case v := <-fba:
                m[v] = m[v] + 1
            case <-tick:
                output <- m
                m = make(map[string]int)
        }
    }
}

/* Receives the map of feedback each tick */
func feedbackOutput(data chan map[string]int){
    for {
        m := <-data;
        for k := range m {
            fmt.Printf("%s had %d feedback hits\n", k, m[k]);
        }
    }
}