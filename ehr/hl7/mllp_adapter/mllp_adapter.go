// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The mllp_adapter binary is a server that accepts HL7 messages over MLLP and
// forwards them to the Cloud HL7 service API.
package main

import (
	"flag"
	
	log "github.com/golang/glog"
	"golang.org/x/net/context"
	"mllp_adapter/healthapiclient"
	"mllp_adapter/mllpreceiver"
	"mllp_adapter/mllpsender"
	"mllp_adapter/monitoring"
	"mllp_adapter/pubsub"
)

var (
	// 2575 is the default port for HL7 over TCP
	// https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml?search=2575
	port               = flag.Int("port", 2575, "Port on which to listen for incoming MLLP connections")
	apiAddrPrefix      = flag.String("api_addr_prefix", "", "Prefix of the Cloud Healthcare API, including scheme and version")
	mllpAddr           = flag.String("mllp_addr", "", "Target address for outgoing MLLP connections")
	receiverIP         = flag.String("receiver_ip", "", "IP address for incoming MLLP connections")
	pubsubProjectID    = flag.String("pubsub_project_id", "", "Project ID that owns the pubsub topic")
	pubsubSubscription = flag.String("pubsub_subscription", "", "Pubsub subscription to read for notifications of new messages")
	hl7ProjectID       = flag.String("hl7_project_id", "", "Project ID that owns the healthcare dataset")
	hl7LocationID      = flag.String("hl7_location_id", "", "ID of Cloud Location where the healthcare dataset is stored")
	hl7DatasetID       = flag.String("hl7_dataset_id", "", "ID of the healthcare dataset")
	hl7StoreID         = flag.String("hl7_store_id", "", "ID the HL7 store inside the healthcare dataset")
	exportStats        = flag.Bool("export_stats", true, "Whether to export stackdriver stats")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	var mon *monitoring.Client
	if *exportStats {
		mon = monitoring.NewClient()
		if err := monitoring.ConfigureExport(ctx, mon); err != nil {
			log.Fatalf("monitoring.ConfigureExport: %v", err)
		}
		// Initial export delay is between 45 and 45+30 seconds
		go func() {
			err := mon.StartExport(ctx, 45, 30)
			log.Fatalf("monitoring.Run: %v", err)
		}()
	}

	apiClient, err := healthapiclient.NewClient(ctx, mon, *apiAddrPrefix, *hl7ProjectID, *hl7LocationID, *hl7DatasetID, *hl7StoreID)
	if err != nil {
		log.Fatalf("healthapiclient.NewClient: %v", err)
	}

	if *pubsubProjectID == "" || *pubsubSubscription == "" {
		log.Infof("Pubsub receiver not specified, starting without")
	} else {
		sender := mllpsender.NewSender(*mllpAddr, mon)
		go func() {
			err := pubsub.Listen(ctx, mon, *pubsubProjectID, *pubsubSubscription, apiClient, sender)
			log.Fatalf("pubsub.Listen: %v", err)
		}()
	}

	if *receiverIP == "" {
		log.Fatalf("Required flag value --receiver_ip not provided")
	}

	receiver, err := mllpreceiver.NewReceiver(*receiverIP, *port, apiClient, mon)
	if err != nil {
		log.Fatalf("NewReceiver: %v", err)
	}
	go func() {
		err := receiver.Run()
		if err != nil {
			log.Fatalf("MLLPReceiver.Run: %v", err)
		}
	}()

	select {}
}
