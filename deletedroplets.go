package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"strings"

	"sync"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

func main() {
	wait := sync.WaitGroup{}
	apiURL := os.Getenv("DIGITALOCEAN_API_URL")
	key := os.Getenv("DIGITALOCEAN_API_KEY")
	if key == "" {
		panic("You must provide a Digital Ocean API Key")
	}
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: key,
	})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)
	if apiURL != "" {
		var err error
		client.BaseURL, err = url.Parse(apiURL)
		if err != nil {
			return
		}
	}

	options := godo.ListOptions{PerPage: 1000}

	droplets, _, err := client.Droplets.List(context.Background(), &options)
	if err != nil {
		return
	}
	for _, dr := range droplets {
		fmt.Println(">", dr.Name)
		if strings.Contains(dr.Name, "gofn") {
			action, _, err := client.DropletActions.Shutdown(context.Background(), dr.ID)
			if err != nil {
				// Power off force Shutdown
				action, _, err = client.DropletActions.PowerOff(context.Background(), dr.ID)
				if err != nil {
					return
				}
			}
			wait.Add(1)
			go func() {
				act := action
				drID := dr.ID
				quit := make(chan struct{})
				errs := make(chan error, 1)
				ac := make(chan *godo.Action, 1)
				go func() {
					for {
						//running shutdown...
						select {
						case <-quit:
							return
						default:
							d, _, err := client.DropletActions.Get(context.Background(), drID, act.ID)
							if err != nil {
								errs <- err
								fmt.Println(err.Error())
								return
							}
							if d.Status == "completed" {
								ac <- d
								return
							}
						}
					}
				}()
				select {
				case action = <-ac:
					_, err = client.Droplets.Delete(context.Background(), drID)
					if err != nil {
						fmt.Println(err.Error())
					}
					wait.Done()
					return
				case err = <-errs:
					wait.Done()
					return
				}
			}()
		}
	}
	wait.Wait()
}
