// Load_config prints the merged and parse config to stdout.
// TODO: delete this binary.
package main

import (
	"fmt"
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

var path = flag.String("config_path", "", "Path to config")

func main() {
	flag.Parse()

	b, err := config.LoadBytes(*path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}
