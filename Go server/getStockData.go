package hello

import (
	"fmt"
	"net/http"
	"oanda"
)

var access_token string = "<api-token>"

func init() {
	http.HandleFunc("/test", test)
}

func test(w http.ResponseWriter, r *http.Request) {
	client, err := oanda.NewFxPracticeClient(access_token, r)
	if err != nil {
		panic(err)
	}

	client.SelectAccount("api-account")

	prices, err := client.PollPrices("eur_usd", "gbp_usd")
	if err != nil {
		panic(err)
	}

	for key, value := range prices {
   		fmt.Fprint(w, key + " ", value.Time, " ", value.Bid, "\n")
	}

}
