// Load_config prints the merged, parsed and validated config to stdout.
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
		log.Fatalf("failed to load cosnfig to bytes: %v", err)
	}

	fmt.Println(string(b))
}
