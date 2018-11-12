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

export const environment = {
  production: false,
  clientId: '',
  // System configuration for which FHIR store is being used for this demo. A
  // production system would require a more dynamic solution. This is overriden
  // by environment.prod.ts.
  fhirEndpoint: {
    baseURL: '',
    project: '',
    location: '',
    dataset: '',
    fhirStore: '',
  }
};

/*
 * In development mode, for easier debugging, you can ignore zone related
 * error stack frames such as `zone.run`/`zoneDelegate.invokeTask` by
 * importing the below file. Don't forget to comment it out in production mode
 * because it will have a performance impact when errors are thrown
 */
// import 'zone.js/dist/zone-error';  // Included with Angular CLI.
