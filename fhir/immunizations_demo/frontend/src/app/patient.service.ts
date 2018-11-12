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
import {Injectable} from '@angular/core';
import {get} from 'lodash';
import {from, Observable} from 'rxjs';
import {map, publishReplay, refCount} from 'rxjs/operators';
import {ResourceService} from './resource.service';

/**
 * A service for retrieving the "logged in" patient.
 */
@Injectable({providedIn: 'root'})
export class PatientService {
  private patient$: Observable<fhir.Patient>|undefined;

  constructor(private readonly resourceService: ResourceService) {}

  /**
   * Retrieve the "logged in" patient.
   */
  getPatient(): Observable<fhir.Patient> {
    if (this.patient$) {
      return this.patient$;
    }
    // We have set up a patient with the identifier 'demo-patient' to simulate
    // a logged in user. In a production application the identifier would come
    // from an auth server.
    this.patient$ = from(this.resourceService.searchResource('Patient', {
                      identifier: 'demo-patient',
                    }))
                        .pipe(
                            map(bundle => get(bundle, 'entry[0].resource')),
                            // Cache the response
                            publishReplay(1),
                            refCount(),
                        );
    return this.patient$;
  }
}
