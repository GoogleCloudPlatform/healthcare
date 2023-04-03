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
import {ComponentFixture, TestBed, waitForAsync} from '@angular/core/testing';
import {ReactiveFormsModule} from '@angular/forms';
import {MatLegacyAutocompleteModule} from '@angular/material/autocomplete';
import {MatDatepickerModule} from '@angular/material/datepicker';
import * as moment from 'moment';
import {of} from 'rxjs';

import {createResourceServiceSpy, resourceServiceSpyProvider} from '../../../test/resource-service-spy';
import {createTestTravelQuestionnaireResponse} from '../../../test/travel-plan';
import {createTestScheduler} from '../../../test/util';
import {ISO_DATE} from '../../constants';
import {PatientService} from '../../patient.service';
import {TravelPlan} from '../travel-plan';

import {TravelFormComponent} from './travel-form.component';

describe('TravelFormComponent', () => {
  let component: TravelFormComponent;
  let fixture: ComponentFixture<TravelFormComponent>;
  const scheduler = createTestScheduler();
  const testFormValues = {
    destination: 'test destination',
    departureDate: moment(),
    returnDate: moment().add(10, 'days'),
  };
  const testPatient = {
    id: '1',
  };
  const travelPlanQuestionnaire: fhir.Questionnaire = {
    id: 'test-questionnaire',
    status: 'final',
  };

  const resourceServiceSpy = createResourceServiceSpy();

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          imports: [
            MatDatepickerModule,
            MatLegacyAutocompleteModule,
            ReactiveFormsModule,
          ],
          declarations: [TravelFormComponent],
          providers: [
            resourceServiceSpyProvider(resourceServiceSpy),
            {
              provide: PatientService,
              useValue: jasmine.createSpyObj(
                  'PatientService', {getPatient: of(testPatient)}),
            },
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(TravelFormComponent);
    component = fixture.componentInstance;

    component.questionnaire = travelPlanQuestionnaire;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should update the form with a travel plan', () => {
    const resp = createTestTravelQuestionnaireResponse(
        testFormValues.destination, testFormValues.departureDate,
        testFormValues.returnDate);
    component.travelPlan = new TravelPlan(resp);

    component.ngOnChanges();

    const formValue = component.form.value;
    expect(formValue.destination).toEqual(testFormValues.destination);
    expect(toDateString(formValue.departureDate))
        .toEqual(toDateString(testFormValues.departureDate));
    expect(toDateString(formValue.returnDate))
        .toEqual(toDateString(testFormValues.returnDate));
  });

  it('should reset the form when no travel plan is specified', () => {
    component.form.setValue(testFormValues);

    component.travelPlan = null;
    component.ngOnChanges();

    const data = component.form.value;
    expect(data.departureDate).not.toBe(testFormValues.departureDate);
    expect(data.returnDate).not.toBe(testFormValues.departureDate);
    expect(data.destination).toEqual('');
  });

  it('emits a cancel event', () => {
    scheduler.run(({expectObservable}) => {
      scheduler.schedule(() => component.cancel(), 1);

      expectObservable(component.cancelled).toBe('-e', {e: undefined});
    });
  });

  it('creates a new travel plan', async () => {
    component.form.setValue(testFormValues);

    const subscription = jasmine.createSpy();
    component.saved.subscribe(subscription);

    await component.submit();

    expect(subscription).toHaveBeenCalled();

    expect(resourceServiceSpy.createResource).toHaveBeenCalledWith({
      resourceType: 'QuestionnaireResponse',
      author: {reference: `Patient/${testPatient.id}`},
      item: [
        {answer: [{valueString: testFormValues.destination}], linkId: '1'}, {
          item: [
            {
              answer:
                  [{valueDate: testFormValues.departureDate.format(ISO_DATE)}],
              linkId: '2.1'
            },
            {
              answer: [{valueDate: testFormValues.returnDate.format(ISO_DATE)}],
              linkId: '2.2'
            }
          ],
          linkId: '2'
        }
      ],
      questionnaire: {reference: `Questionnaire/${travelPlanQuestionnaire.id}`},
      status: 'completed',
      subject: {reference: `Patient/${testPatient.id}`}
    });
  });

  it('updates an existing travel plan', async () => {
    const resp = createTestTravelQuestionnaireResponse(
        testFormValues.destination, testFormValues.departureDate,
        testFormValues.returnDate);
    component.travelPlan = new TravelPlan(resp);

    component.ngOnChanges();

    const destination = 'Updated test destination';
    component.form.patchValue({
      destination,
    });

    await component.submit();

    expect(resourceServiceSpy.saveResource).toHaveBeenCalled();
    const savedResource = resourceServiceSpy.saveResource.calls.argsFor(0)[0];
    expect(savedResource.item[0]).toEqual({
      answer: [{valueString: destination}],
      linkId: '1',
    });
  });

  function toDateString(m: moment.Moment): string {
    return m.format(ISO_DATE);
  }
});
