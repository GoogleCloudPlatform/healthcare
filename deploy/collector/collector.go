/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package collector provides handlers to collect audit data.
package collector

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/healthcare/deploy/collector/processors"
	"google3/third_party/golang/appengine/appengine"
	"google3/third_party/golang/appengine/log/log"
	"google3/third_party/golang/cloud/storage/storage"
	"google3/third_party/golang/gorilla/mux/mux"
)

var (
	initTime  = time.Now()
	apiClient *processors.AuditAPIClient
)

func init() {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.Handle("/debug/clear_all", handlerFunc(clearAll))
	r.Handle("/debug/dump_all", handlerFunc(dumpAll))
	r.Handle("/processors/update_all", handlerFunc(updateAll))
	http.Handle("/", r)
	apiClient = processors.NewAuditAPIClient()
}

type handlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// ServeHTTP is an interface function for http.Handler.
func (f handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if err := f(ctx, w, r); err != nil {
		log.Errorf(ctx, "%v", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	out := fmt.Sprint("Welcome to audit data collector! %v\n", time.Since(initTime))
	log.Infof(ctx, "Serving: %s", out)
	fmt.Fprint(w, out)
}

// Calls processers to update audit data.
func updateAll(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	log.Infof(ctx, "Serving: update all")

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	gcsClient := processors.NewGCSClient(os.Getenv("CLOUD_STORAGE_BUCKET"), client)
	if err := processors.UpdateProjectConfig(ctx, os.Getenv("PROJECT_CONFIG_YAML_PATH"), apiClient, gcsClient); err != nil {
		return err
	}
	return nil
}

// A test function that outputs all audit data to webpage in text.
func dumpAll(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	log.Infof(ctx, "Serving: dump all")
	fmt.Fprint(w, apiClient.DebugDumpAll())
	return nil
}

// A test function that clears all stored audit data in AuditAPIClient.
func clearAll(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	output := "clear all"
	log.Infof(ctx, "Serving: %s", output)
	fmt.Fprint(w, output)
	return nil
}
