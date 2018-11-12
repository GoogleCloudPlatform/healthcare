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

import {CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {async, ComponentFixture, TestBed} from '@angular/core/testing';
import {ReactiveFormsModule} from '@angular/forms';
import {MatDatepickerModule, MatFormFieldModule, MatInputModule, MatSelectModule} from '@angular/material';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import * as moment from 'moment';

import {environment} from '../../../environments/environment';
import {createTestImmunization, createTestReaction} from '../../../test/immunization';
import {createResourceServiceSpy, resourceServiceSpyProvider} from '../../../test/resource-service-spy';
import {createTestScheduler, resetSpyObj} from '../../../test/util';
import {ISO_DATE} from '../../constants';
import {FHIR_STORE} from '../../fhir-store';
import {Immunization, ImmunizationReaction, ImmunizationReason} from '../immunization';
import {ImmunizationFormComponent} from './immunization-form.component';

describe('ImmunizationFormComponent', () => {
  let component: ImmunizationFormComponent;
  let fixture: ComponentFixture<ImmunizationFormComponent>;
  const epoch = moment('1970-01-01');
  const testFormValues = {
    name: 'Test vaccine',
    date: epoch,
    expirationDate: epoch,
    reaction: ImmunizationReaction.AnaphalacticShock,
    reason: ImmunizationReason.Occupational,
    note: 'Note',
  };
  const resourceServiceSpy = createResourceServiceSpy();
  const scheduler = createTestScheduler();

  beforeEach(async(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          imports: [
            MatDatepickerModule,
            ReactiveFormsModule,
            MatSelectModule,
            MatFormFieldModule,
            MatInputModule,
            NoopAnimationsModule,
          ],
          declarations: [ImmunizationFormComponent],
          providers: [
            {provide: FHIR_STORE, useValue: environment.fhirEndpoint},
            resourceServiceSpyProvider(resourceServiceSpy),
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ImmunizationFormComponent);
    component = fixture.componentInstance;

    let id = 0;
    resetSpyObj(resourceServiceSpy);
    resourceServiceSpy.createResource.and.callFake((r: fhir.Resource) => {
      r = JSON.parse(JSON.stringify(r));
      r.id = id.toString();
      id += 1;
      return Promise.resolve(r);
    });
    resourceServiceSpy.saveResource.and.callFake(
        (r: fhir.Resource) => Promise.resolve(r));

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should update the form with an immunization', () => {
    const vaccineCode = 'Test immunization';
    const immNote = 'Not a real immunization';
    const imm = new Immunization(createTestImmunization(
        vaccineCode, ImmunizationReason.Occupational, immNote));
    component.immunization = imm;
    const reaction: fhir.Observation =
        createTestReaction(ImmunizationReaction.AnaphalacticShock);
    component.reaction = reaction;

    component.ngOnChanges();

    const formValue = component.form.value;
    expect((formValue.date as moment.Moment).isSame(epoch));
    expect((formValue.expirationDate as moment.Moment).isSame(epoch));
    expect(formValue).toEqual(jasmine.objectContaining({
      name: vaccineCode,
      reaction: ImmunizationReaction.AnaphalacticShock,
      reason: ImmunizationReason.Occupational,
      note: immNote,
    }));
  });

  it('should reset the form when no immunization is specified', () => {
    component.form.setValue(testFormValues);

    component.immunization = null;
    component.ngOnChanges();

    const data = component.form.value;
    expect(data.date).not.toEqual(epoch);
    expect(data).toEqual(jasmine.objectContaining({
      name: '',
      expirationDate: undefined,
      reaction: undefined,
      reason: undefined,
      note: undefined,
    }));
  });

  it('emits a cancel event when cancel is called', () => {
    scheduler.run(({expectObservable}) => {
      scheduler.schedule(() => component.cancel(), 1);

      expectObservable(component.cancelled).toBe('-e', {e: undefined});
    });
  });

  it('creates a new immunization and reaction when the form is saved in create mode',
     async () => {
       component.form.setValue(testFormValues);

       const subscription = jasmine.createSpy();
       component.saved.subscribe(subscription);

       await component.submit();

       expect(subscription).toHaveBeenCalled();

       expect(resourceServiceSpy.createResource).toHaveBeenCalledWith({
         resourceType: 'Immunization',
         status: 'completed',
         notGiven: false,
         date: testFormValues.date.toISOString(),
         expirationDate: testFormValues.expirationDate.format(ISO_DATE),
         vaccineCode: {text: testFormValues.name},
         patient: {},
         primarySource: true,
         explanation: {
           reason: [{
             coding: [{
               system: 'http://snomed.info/sct',
               code: testFormValues.reason,
             }],
           }],
         },
         note: [{text: testFormValues.note}],
       });

       expect(resourceServiceSpy.saveResource)
           .toHaveBeenCalledWith(jasmine.objectContaining({
             resourceType: 'Observation',
             id: `0-reaction`,
             valueCodeableConcept: {
               coding: [{display: testFormValues.reaction}],
             },
           }));
     });

  it('updates an existing immunization and reaction when the form is saved in edit mode',
     async () => {
       const fhirImm = createTestImmunization(
           testFormValues.name, testFormValues.reason, testFormValues.note);
       fhirImm.id = '0';
       component.immunization = new Immunization(fhirImm);

       const reaction =
           createTestReaction(ImmunizationReaction.AnaphalacticShock);
       component.immunization.reactionID = reaction.id =
           `${fhirImm.id}-reaction`;
       component.reaction = reaction;

       component.ngOnChanges();

       const note = 'Updated note';
       const updatedReaction = ImmunizationReaction.Seizure;
       component.form.patchValue({
         note,
         reaction: updatedReaction,
       });

       await component.submit();

       expect(resourceServiceSpy.saveResource)
           .toHaveBeenCalledWith(jasmine.objectContaining({
             note: [{text: note}],
           }));

       expect(resourceServiceSpy.saveResource)
           .toHaveBeenCalledWith(jasmine.objectContaining({
             valueCodeableConcept: {
               coding: [{display: updatedReaction}],
             },
           }));
     });

  it('removes an existing reaction when the immunization is saved with no reaction',
     async () => {
       const fhirImm = createTestImmunization(
           testFormValues.name, testFormValues.reason, testFormValues.note);
       fhirImm.id = '0';
       component.immunization = new Immunization(fhirImm);

       const reaction =
           createTestReaction(ImmunizationReaction.AnaphalacticShock);
       component.immunization.reactionID = reaction.id =
           `${fhirImm.id}-reaction`;
       component.reaction = reaction;

       component.ngOnChanges();

       component.form.patchValue({
         reaction: undefined,
       });

       await component.submit();

       expect(resourceServiceSpy.saveResource).toHaveBeenCalled();
       const savedImm: fhir.Immunization =
           resourceServiceSpy.saveResource.calls.argsFor(0)[0];
       expect(savedImm.reaction).toBeUndefined();

       expect(resourceServiceSpy.deleteResource)
           .toHaveBeenCalledWith('Observation', reaction.id);
     });

  it('adds a reaction when the form is saved with a reaction', async () => {
    const fhirImm = createTestImmunization(
        testFormValues.name, testFormValues.reason, testFormValues.note);
    fhirImm.id = '0';
    component.immunization = new Immunization(fhirImm);

    component.ngOnChanges();

    component.form.patchValue({
      reaction: ImmunizationReaction.Seizure,
    });

    await component.submit();

    expect(resourceServiceSpy.saveResource)
        .toHaveBeenCalledWith(jasmine.objectContaining({
          resourceType: 'Observation',
          id: '0-reaction',
          valueCodeableConcept: {
            coding: [{display: ImmunizationReaction.Seizure}],
          },
        }));

    expect(resourceServiceSpy.saveResource)
        .toHaveBeenCalledWith(jasmine.objectContaining({
          resourceType: 'Immunization',
          reaction: [
            {
              detail: {
                reference: 'Observation/0-reaction',
              },
              reported: true,
            },
          ],
        }));
  });
});
