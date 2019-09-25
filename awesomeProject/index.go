package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nlopes/slack"
	"log"
	"os"
	"strconv"
	"strings"
)

type Order struct{
	pizzatype string
	pizzasize int
	address string
}

type Order_table struct {
	id int
	pizzatype string
	pizzasize int
	address string
	user string
}

var db *sql.DB

func init() {
	var err error
	db, err := sql.Open("mysql", "pizzabot:[password]@tcp(host1.flynet.pro)/pizzabot")
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	defer db.Close()
}

func orderIndex() {

	rows, err := db.Query("SELECT * FROM orders")
	if err != nil {
		return;
	}
	defer rows.Close()

	orders := make([]*Order_table, 0)
	for rows.Next() {
		order := new(Order_table)
		err := rows.Scan(&order.id, &order.pizzatype, &order.pizzasize, &order.address, &order.user)
		if err != nil {
			return
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return
	}

	for _, order := range orders {
		fmt.Printf("%s, %s, %s, £%.2f\n", order.pizzatype, order.address, order.user, order.pizzasize)
	}
}

func main() {
	orderIndex()

	api := slack.New(
		"[token]",
		slack.OptionDebug(true),
		slack.OptionLog(
			log.New(os.Stdout, "slack-bot: ",
				log.Lshortfile|log.LstdFlags)),
	)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	profileStates := make(map[string]int)
	temporders := make(map[string]Order)

	Loop:
		for {
			select {
			case msg := <-rtm.IncomingEvents:
				fmt.Print("Event Received: ")

				switch ev := msg.Data.(type) {
					case *slack.MessageEvent:
						text := ev.Text
						//text = strings.TrimSpace(text) //whynot?
						text = strings.ToLower(text)

						if (strings.Contains(text, "/start")) {
							profileStates[ev.User] = 1;
							rtm.SendMessage(rtm.NewOutgoingMessage("Hi! Use \n" +
								"/create to begin your order \n " +
								"/status all - to get all your orders \n " +
								"/status [number] - to get info about [number] order \n" +
								"don't forger about space before slash!", ev.Channel))
							break;
						}
						if (strings.Contains(text, "/create")) {
							profileStates[ev.User] = 2;
							rtm.SendMessage(rtm.NewOutgoingMessage("New order [1/3] - Enter pizza type: [string]", ev.Channel))
							break;
						}
						if(profileStates[ev.User] >= 2 && profileStates[ev.User] <= 10) {

							if (profileStates[ev.User] == 2) {
								temporders[ev.User] = Order{
									pizzatype: text,
									pizzasize: temporders[ev.User].pizzasize,
									address: temporders[ev.User].address,
								}
								profileStates[ev.User] = 3;
								rtm.SendMessage(rtm.NewOutgoingMessage("New order [2/3] - Enter pizza size: [int]", ev.Channel))
								break;
							}

							if (profileStates[ev.User] == 3) {
								var i, err = strconv.Atoi(text)

								if err != nil {
									rtm.SendMessage(rtm.NewOutgoingMessage("New order [2/3] - Enter pizza size: [int]", ev.Channel))
									break;
								}

								temporders[ev.User] = Order{
									pizzatype: temporders[ev.User].pizzatype,
									pizzasize: i,
									address: temporders[ev.User].address,
								}

								profileStates[ev.User] = 4;
								rtm.SendMessage(rtm.NewOutgoingMessage("New order [3/3] - Enter address: [string]", ev.Channel))
								break;
							}

							if (profileStates[ev.User] == 4) {
								temporders[ev.User] = Order{
									pizzatype: temporders[ev.User].pizzatype,
									pizzasize: temporders[ev.User].pizzasize,
									address: text,
								}
								var txt = strconv.Itoa(temporders[ev.User].pizzasize);
								rtm.SendMessage(rtm.NewOutgoingMessage("New order [confirm]: yes/no \n *" +
									temporders[ev.User].pizzatype + "\n *"+
									temporders[ev.User].address + "\n *" +
									txt, ev.Channel))
								profileStates[ev.User] = 5
								break;
							}
							if (profileStates[ev.User] == 4) {
								if (text == "yes") {
									//To-Do: MYSQL connection

								}
								if (text == "no") {
									profileStates[ev.User] = 1
									rtm.SendMessage(rtm.NewOutgoingMessage("Ok, aborted \n\n", ev.Channel))

									rtm.SendMessage(rtm.NewOutgoingMessage("Hi! Use \n" +
										"/create to begin your order \n " +
										"/status all - to get all your orders \n " +
										"/status [number] - to get info about [number] order \n" +
										"don't forger about space before slash!", ev.Channel))
								}
							}
						}
					case *slack.RTMError:
						fmt.Printf("Error: %s\n", ev.Error())

					case *slack.InvalidAuthEvent:
						fmt.Printf("Invalid credentials")
						break Loop

					default:
						// Take no action
				}
			}
		}
}