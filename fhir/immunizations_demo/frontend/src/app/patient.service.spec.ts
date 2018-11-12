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
import {async, TestBed} from '@angular/core/testing';
import {createResourceServiceSpy, resourceServiceSpyProvider} from '../test/resource-service-spy';
import {PatientService} from './patient.service';

describe('PatientService', () => {
  const demoPatient: fhir.Patient = {
    resourceType: 'Patient',
    id: '1',
  };
  const patientSearchResult: fhir.Bundle = {
    type: 'searchset',
    entry: [{
      resource: demoPatient,
    }],
  };
  const resourceServiceSpy = createResourceServiceSpy();

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      providers: [
        PatientService,
        resourceServiceSpyProvider(resourceServiceSpy),
      ],
    });
  }));

  it('should retrieve the patient', async () => {
    resourceServiceSpy.searchResource.and.returnValue(
        Promise.resolve(patientSearchResult));

    const service: PatientService = TestBed.get(PatientService);
    const patient = await service.getPatient().toPromise();

    expect(patient).toEqual(demoPatient);
    expect(resourceServiceSpy.searchResource).toHaveBeenCalledWith('Patient', {
      identifier: 'demo-patient',
    });
  });
});
