package main

import (
    "net"
    "bufio"
    "fmt"
    "strings"
    "strconv"
    "encoding/json"

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
        sline = strings.TrimSuffix(sline, "\r")

        //fmt.Printf(sline)
        fmt.Printf("%s received from %v\n", sline, client.RemoteAddr())
        fba <- sline
    }
}

type feedbackMapsTypeBecauseGoHasNoPairsWhichIsDumb struct {
    Hitmap map[string]string
    Scoremap map[string]string
}

type BestUser struct {
    Username string
    Score string
}

/* Thread to accumulate all feedback from clients into a map each tick */
func feedbackAccumulator(fba chan string, tick chan dataType, output chan feedbackMapsTypeBecauseGoHasNoPairsWhichIsDumb){
    var m map[string]string = make(map[string]string)
    var hs map[string]string = make(map[string]string)
    var hstime map[string]int = make(map[string]int)
    var best BestUser = BestUser{"anon", "0"}

    for {
        select {
            case v := <-fba:
                vdata := strings.Split(v, ":")
                fmt.Println(vdata) 
                if(vdata[0] == "move"){
                    i, _ := strconv.Atoi(m[vdata[1]])
                    m[vdata[1]] = strconv.Itoa(i + 1)
                } else if(vdata[0] == "score"){
                    hs[vdata[1]] = vdata[2]
                    hstime[vdata[1]] = 50

                    val, _ := strconv.ParseFloat(vdata[2], 64)
                    newVal := int(val)
                    oldVal, _ := strconv.Atoi(best.Score)

                    if(newVal > oldVal) {
                        best = BestUser {vdata[1], vdata[2]}
                    }
                    fmt.Println(best.Score)
                } else{
                    fmt.Println(vdata) 
                }
            case tick := <-tick:
                for nick := range hstime {
                    hstime[nick] = hstime[nick] - 1;
                    if(hstime[nick] == 0){
                        delete(hs, nick)
                    }
                }

                output <- feedbackMapsTypeBecauseGoHasNoPairsWhichIsDumb{m,hs}
                m = make(map[string]string)
                m["tick"] = strconv.FormatFloat(tick.val, 'f', -1, 64)
                m["mean"] = strconv.FormatFloat(tick.mean, 'f', -1, 64)
                m["sd"] = strconv.FormatFloat(tick.sd, 'f', -1, 64)
                m["bestUser"] = best.Username
                m["bestScore"] = best.Score
        }
    }
}

/* Receives the map of feedback each tick */
func feedbackOutput(data chan feedbackMapsTypeBecauseGoHasNoPairsWhichIsDumb, outputWeb chan feedbackMapsTypeBecauseGoHasNoPairsWhichIsDumb){
    for {
        m := <-data;
        for k := range m.Hitmap {
            if(k != "tick" && k != "mean" && k != "sd" && 
                k != "bestScore" && k != "bestUser") { 
                fmt.Printf("%s had %s feedback hits\n", k, m.Hitmap[k]); 
            }
        }

        select {
            case outputWeb<-m: 
            default:
        }
    }
}

func socketHandler(data chan feedbackMapsTypeBecauseGoHasNoPairsWhichIsDumb) websocket.Handler {
  return func(ws *websocket.Conn) {
    for { 
        i := <-data;
        m := i.Hitmap;
        if(m["1"] == "") { m["1"] = "0" }
        if(m["-1"] == "") { m["-1"] = "0" }
        //var res = m["tick"] + ", " + m["1"] + ", " + m["-1"] + ", " + m["mean"] + ", " + m["sd"]

        j, _ := json.Marshal(i)
        _, err := ws.Write([]byte(string(j)))

        //_, err := ws.Write([]byte(res))
        if err != nil {
            break
        }
    }
  }
}
