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
import {fakeAsync, tick} from '@angular/core/testing';
import * as moment from 'moment';
import {of} from 'rxjs';

import {createTestImmunization} from '../test/immunization';
import {ImmunizationServiceStub} from '../test/immunization-service-stub';
import {createResourceServiceSpy} from '../test/resource-service-spy';
import {resetSpyObj} from '../test/util';
import {Immunization} from './immunizations/immunization';
import {PatientService} from './patient.service';
import {PredictionService} from './prediction.service';

describe('PredictionService', () => {
  const testPatient = {
    id: '1',
  };
  const yellowFeverAssessment: fhir.RiskAssessment = {
    resourceType: 'RiskAssessment',
    id: '1',
    status: 'final',
    prediction: [{
      outcome: {
        coding: [{
          display: 'Yellow Fever',
        }]
      },
    }]
  };
  const meningitisAssessment: fhir.RiskAssessment = {
    resourceType: 'RiskAssessment',
    id: '2',
    status: 'final',
    prediction: [{
      outcome: {
        coding: [{
          display: 'Meningitis',
        }]
      },
    }]
  };

  const searchResults = {
    entry: [{
      resource: yellowFeverAssessment,
    }],
  };
  let immunizationServiceStub: ImmunizationServiceStub;
  const patientServiceSpy =
      jasmine.createSpyObj<PatientService>('PatientService', ['getPatient']);
  const resourceServiceSpy = createResourceServiceSpy();

  beforeAll(fakeAsync(() => {
    immunizationServiceStub = new ImmunizationServiceStub();
  }));

  beforeEach(() => {
    resetSpyObj(patientServiceSpy);
    resetSpyObj(resourceServiceSpy);
  });

  it('should emit a list of risk assessments on subscribe', fakeAsync(() => {
       const resourceServiceSpy = createResourceServiceSpy();

       patientServiceSpy.getPatient.and.returnValue(of(testPatient));
       resourceServiceSpy.searchResource.and.returnValue(of(searchResults));

       const service = new PredictionService(
           resourceServiceSpy, patientServiceSpy, immunizationServiceStub);

       const observer = jasmine.createSpy();

       const sub = service.predictions$.subscribe(observer);
       tick(1);
       expect(observer).toHaveBeenCalledWith([yellowFeverAssessment]);

       sub.unsubscribe();
     }));

  it('should emit a list of risk assessments after the poll interval',
     fakeAsync(() => {
       patientServiceSpy.getPatient.and.returnValue(of(testPatient));
       resourceServiceSpy.searchResource.and.returnValue(of(searchResults));

       const service = new PredictionService(
           resourceServiceSpy, patientServiceSpy, immunizationServiceStub);

       // No-op sub to start the polling
       const sub1 = service.predictions$.subscribe(() => {});
       tick();

       const observer = jasmine.createSpy('poll listener');
       const sub2 = service.predictions$.subscribe(observer);
       tick(15000);

       // Once because the first result is cached, second time when the poll
       // happens.
       expect(observer).toHaveBeenCalledTimes(2);
       expect(observer).toHaveBeenCalledWith([yellowFeverAssessment]);

       sub1.unsubscribe();
       sub2.unsubscribe();
     }));

  it('should filter the risk assessments by received immunizations',
     fakeAsync(() => {
       const resourceServiceSpy = createResourceServiceSpy();

       patientServiceSpy.getPatient.and.returnValue(of(testPatient));
       resourceServiceSpy.searchResource.and.returnValue(of({
         entry: [
           {resource: yellowFeverAssessment},
           {resource: meningitisAssessment},
         ]
       }));

       const service = new PredictionService(
           resourceServiceSpy, patientServiceSpy, immunizationServiceStub);

       const imm = new Immunization(createTestImmunization('Yellow Fever'));
       imm.expirationDate = moment.invalid();
       immunizationServiceStub.immunizationSubject.next([imm]);

       const observer = jasmine.createSpy();

       const sub = service.predictions$.subscribe(observer);
       tick(1);
       expect(observer).toHaveBeenCalledWith([meningitisAssessment]);

       sub.unsubscribe();
     }));
});
