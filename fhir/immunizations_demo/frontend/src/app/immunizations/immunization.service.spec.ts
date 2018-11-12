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
import {async, fakeAsync, flush, tick} from '@angular/core/testing';
import {BehaviorSubject, of} from 'rxjs';

import {createTestImmunization} from '../../test/immunization';
import {createResourceServiceSpy} from '../../test/resource-service-spy';
import {expectAsyncToThrowError, resetSpyObj} from '../../test/util';

import {Immunization} from './immunization';
import {ImmunizationService} from './immunization.service';

describe('ImmunizationService', () => {
  const resourceServiceSpy = createResourceServiceSpy();
  let service: ImmunizationService;
  const testImm = new Immunization({
    ...createTestImmunization(),
    id: '1',
  });
  let immunizationListener: BehaviorSubject<Immunization[]>;

  beforeEach(async(() => {
    resetSpyObj(resourceServiceSpy);

    resourceServiceSpy.searchResource.and.returnValue(
        of({
          entry: [{
            resource: testImm.toFHIR(),
          }],
        }).toPromise());
    service = new ImmunizationService(resourceServiceSpy);

    immunizationListener = new BehaviorSubject([] as Immunization[]);
    service.immunizations$.subscribe(immunizationListener);
  }));

  it('fetch all the patient\'s immunizations', fakeAsync(() => {
       expectImmArrayToEqual(immunizationListener.getValue(), [testImm]);
     }));

  it('delete the immunization', fakeAsync(async () => {
       const toDelete = immunizationListener.getValue()[0];
       await service.remove(toDelete);

       expect(resourceServiceSpy.deleteResource)
           .toHaveBeenCalledWith(jasmine.objectContaining({
             resourceType: 'Immunization',
             id: toDelete.toFHIR().id,
           }));
     }));

  it('should throw an error for immunizations not in the search list',
     fakeAsync(async () => {
       expectAsyncToThrowError(() => service.remove(testImm), /not found/);

       expect(resourceServiceSpy.deleteResource).not.toHaveBeenCalled();
     }));

  it('should add a new dose', fakeAsync(async () => {
       const fhirImm = createTestImmunization();
       const immunization = new Immunization(fhirImm);
       resourceServiceSpy.executeBatch.and.callFake((b: fhir.Bundle) => b);

       await service.addNewDose(immunization);

       const bundle = resourceServiceSpy.executeBatch.calls.argsFor(0)[0];
       expect(bundle.entry![0]).toEqual(jasmine.objectContaining({
         request: {
           method: 'PUT',
           url: `Immunization/${fhirImm.id}`,
         },
       }));
       expect(bundle.entry![0].resource).toEqual(jasmine.objectContaining({
         meta: getUpdatedMeta(),
       }));

       expect(bundle.entry![1]).toEqual(jasmine.objectContaining({
         request: {
           method: 'POST',
           url: 'Immunization',
         },
       }));
     }));

  function getUpdatedMeta(): fhir.Meta {
    return {
      tag: [
        {
          system: 'http://hl7.org/fhir/tag',
          code: 'updated',
        },
      ],
    };
  }

  function expectImmArrayToEqual(
      imms1: Immunization[], imms2: Immunization[]): void {
    expect(imms1.map(i => i.toFHIR())).toEqual(imms2.map(i => i.toFHIR()));
  }
});
