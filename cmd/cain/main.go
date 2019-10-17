package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/r3labs/sse"

	"github.com/spf13/cobra"

	sseclient "github.com/bialas1993/cain/pkg/http/sse"
	"github.com/bialas1993/cain/pkg/logger"
)

var buffer bytes.Buffer
var log = logger.New()

var rootCmd = &cobra.Command{
	Use:   "cain",
	Short: "Load generator",
	Long:  `Load generator to kill SSE service`,
	Run:   Load,
}

func init() {
	godotenv.Load()
	rootCmd.Flags().StringP("uri", "u", "", "address to test")
	rootCmd.Flags().IntP("delay", "d", 150, "delay for add new connection [miliseconds]")
	rootCmd.Flags().IntP("limit", "l", 0, "connections limit (default 0)")
}

func main() {
	rootCmd.Execute()
}

func Load(cmd *cobra.Command, args []string) {
	uri, err := cmd.Flags().GetString("uri")
	delay, _ := cmd.Flags().GetInt("delay")
	limit, _ := cmd.Flags().GetInt("limit")

	if err != nil || len(uri) == 0 {
		fmt.Println("Uri is not set.")
		os.Exit(0)
	}

	fmt.Printf("Address: %s, connections limit: %d, delay new connection: %d\n", uri, limit, delay)

	openedConnections := 0
	clients := make(chan int, 1)
	ticker := time.NewTicker(time.Duration(delay) * time.Millisecond)
	events := make(chan *sse.Event)
	connectFail := false

	defer func() {
		close(clients)
		close(events)
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for {
		select {
		case c := <-clients:
			fmt.Printf("clients: %+v\n", c)

			if err := sseclient.NewClient(uri, events); err != nil {
				fmt.Errorf("Can not create connection, opened: %d", openedConnections)
				connectFail = true
			}
			break
		case <-ticker.C:
			if !connectFail && (openedConnections < limit || limit == 0) {
				openedConnections++

				go func(c int) { clients <- c }(openedConnections)
				continue
			}

			ticker.Stop()
			break

		case event := <-events:
			fmt.Printf("|")
			if len(event.ID) > 0 {
				go log.Write(&logger.Log{event, openedConnections})
			}
			break
		case <-c:
			fmt.Println("\nClosing..\n")
			os.Exit(0)
		}
	}
}