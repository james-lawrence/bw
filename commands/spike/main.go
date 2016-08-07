package main

import (
	"bufio"
	"fmt"
	"os"

	"bitbucket.org/jatone/bearded-wookie/packagekit"
	"github.com/kballard/go-shellquote"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	app := kingpin.New("spike", "spike command line for testing functionality")
	quit := app.Command("quit", "quit the application")

	packages := app.Command("packages", "list the packages on the current machine")
	packageFilter := packages.Arg("filter", "uint64 representing the filter to use.").Default("0").Uint64()

	args := make([]string, len(os.Args)-1, (len(os.Args)-1)*2)
	copy(args, os.Args[1:])
	for {
		var (
			command string
			input   string
			err     error
			client  packagekit.Client
			tx      packagekit.Transaction
		)

		if command, err = app.Parse(args); err != nil {
			goto input
		}

		switch command {
		case packages.FullCommand():
			fmt.Println(*packageFilter)

			//packageFilters := map[string]uint64{
			//"FilterNone":         packagekit.FilterNone,
			//"FilterInstalled":    packagekit.FilterInstalled,
			//"FilterNotInstalled": packagekit.FilterNotInstalled,
			//}

			client, err = packagekit.NewClient()
			if err != nil {
				fmt.Println("Error creating client:", err)
			}

			tx, err = client.CreateTransaction()
			if err != nil {
				fmt.Println("Error creating transaction:", err)
			}

			results, err := tx.Packages(packagekit.PackageFilter(*packageFilter))
			if err != nil {
				fmt.Println("Error getting packages:", err)
			}

			fmt.Println(len(results))
		case quit.FullCommand():
			return
		}

	input:
		fmt.Print(">")
		if input, err = reader.ReadString('\n'); err != nil {
			fmt.Println("Scan Error:", err)
			continue
		}
		if args, err = shellquote.Split(input); err != nil {
			fmt.Println("Input Error:", err)
			continue
		}
	}
}
