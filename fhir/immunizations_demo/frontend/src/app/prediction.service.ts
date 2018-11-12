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
import * as moment from 'moment';
import {combineLatest, from, Observable, timer} from 'rxjs';
import {concatMap, map, publishReplay, refCount} from 'rxjs/operators';

import {ImmunizationService} from './immunizations/immunization.service';
import {PatientService} from './patient.service';
import {ResourceService} from './resource.service';

/**
 * A service to return any condition risk predictions that the current patient
 * has.
 */
@Injectable({
  providedIn: 'root',
})
export class PredictionService {
  /**
   * A stream of all the risk assessments for the logged in patient. Updates
   * periodically.
   */
  predictions$: Observable<fhir.RiskAssessment[]>;

  constructor(
      resourceService: ResourceService,
      patientService: PatientService,
      immunizationService: ImmunizationService,
  ) {
    // Retrieves any risk assessments and then continues polling every 15
    // seconds until all subscribers have unsubscribed.
    const immunizedDiseases$ =
        immunizationService.immunizations$.pipe(map(immunizations => {
          const now = moment();
          const validImmunizations = immunizations.filter(
              imm => !imm.expirationDate.isValid() ||
                  imm.expirationDate.isAfter(now));
          return new Set(validImmunizations.map(imm => imm.name));
        }));

    const resourceStream = combineLatest(
        patientService.getPatient(), immunizedDiseases$, timer(0, 15000));

    this.predictions$ = resourceStream.pipe(
        concatMap(
            ([patient, immunized, _time]) =>
                from(resourceService.searchResource(
                         'RiskAssessment',
                         {subject: `Patient/${patient.id}`},
                         ))
                    .pipe(
                        map(bundle => this.parseAndFilterRiskAssessments(
                                bundle, immunized)))),
        publishReplay(1),
        refCount(),
    );
  }

  private parseAndFilterRiskAssessments(
      bundle: fhir.Bundle,
      immunized: Set<string>,
      ): fhir.RiskAssessment[] {
    const entries = bundle.entry || [];
    return entries.map(entry => entry.resource! as fhir.RiskAssessment)
        .filter(riskAssessment => {
          const condition =
              get(riskAssessment, 'prediction[0].outcome.coding[0].display');
          return !immunized.has(condition);
        });
  }
}
