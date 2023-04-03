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
import {MatLegacyFormFieldModule} from '@angular/material/form-field';
import {MatLegacyInputModule} from '@angular/material/input';
import {MatLegacySelectModule} from '@angular/material/select';
import {By} from '@angular/platform-browser';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import * as moment from 'moment';

import {createResourceServiceSpy, resourceServiceSpyProvider} from '../../../test/resource-service-spy';
import {createTestTravelQuestionnaireResponse} from '../../../test/travel-plan';
import {TravelFormComponent} from '../travel-form/travel-form.component';

import {TravelListComponent} from './travel-list.component';

describe('TravelListComponent', () => {
  let component: TravelListComponent;
  let fixture: ComponentFixture<TravelListComponent>;
  const travelQuestionnaire: fhir.Questionnaire = Object.freeze({
    resourceType: 'Questionnaire',
    status: 'active',
    identifier: [{
      value: 'travel-questionnaire',
    }],
  });

  const resourceServiceSpy = createResourceServiceSpy();

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          imports: [
            NoopAnimationsModule,
            MatDatepickerModule,
            MatLegacyAutocompleteModule,
            ReactiveFormsModule,
            MatLegacySelectModule,
            MatLegacyFormFieldModule,
            MatLegacyInputModule,
          ],
          declarations: [TravelListComponent, TravelFormComponent],
          providers: [resourceServiceSpyProvider(resourceServiceSpy)],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(TravelListComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should retrieve the collection of travel plans', async () => {
    const searchBundle: fhir.Bundle = {
      resourceType: 'Bundle',
      type: 'searchset',
      entry: [
        {
          resource: createTestTravelQuestionnaireResponse(),
        },
        {
          resource: travelQuestionnaire,
        }
      ],
    };
    resourceServiceSpy.searchResource.and.returnValue(
        Promise.resolve(searchBundle));

    fixture.detectChanges();
    await fixture.whenStable();

    expect(component.upcomingTravel.length).toBe(1);
  });

  it('should not display responses if the questionnaire could not be retreived',
     async () => {
       const searchBundle: fhir.Bundle = {
         resourceType: 'Bundle',
         type: 'searchset',
         entry: [{
           resource: createTestTravelQuestionnaireResponse(),
         }],
       };
       resourceServiceSpy.searchResource.and.returnValue(
           Promise.resolve(searchBundle));

       fixture.detectChanges();
       await fixture.whenStable();

       expect(component.upcomingTravel.length).toBe(0);
       expect(component.pastTravel.length).toBe(0);
     });

  it('should split off the past travel plans', async () => {
    const travelPlan = createTestTravelQuestionnaireResponse(
        'dest', moment().subtract(2, 'months'));
    const searchBundle: fhir.Bundle = {
      resourceType: 'Bundle',
      type: 'searchset',
      entry: [
        {
          resource: travelPlan,
        },
        {
          resource: travelQuestionnaire,
        }
      ],
    };
    resourceServiceSpy.searchResource.and.returnValue(
        Promise.resolve(searchBundle));

    fixture.detectChanges();
    await fixture.whenStable();

    expect(component.pastTravel.length).toBe(1);
  });

  it('should order travel plans by date', async () => {
    const travelPlan1Name = 'dest1';
    const travelPlan1 = createTestTravelQuestionnaireResponse(
        travelPlan1Name, moment().subtract(2, 'years'));
    travelPlan1.id = '1';
    const travelPlan2Name = 'dest2';
    const travelPlan2 = createTestTravelQuestionnaireResponse(
        travelPlan2Name, moment().subtract(1, 'years'));
    travelPlan2.id = '2';
    const searchBundle: fhir.Bundle = {
      resourceType: 'Bundle',
      type: 'searchset',
      entry: [
        {resource: travelPlan1},
        {resource: travelPlan2},
        {resource: travelQuestionnaire},
      ],
    };
    resourceServiceSpy.searchResource.and.returnValue(
        Promise.resolve(searchBundle));

    fixture.detectChanges();
    await fixture.whenStable();

    expect(component.pastTravel[0].destination).toBe(travelPlan2Name);
    expect(component.pastTravel[1].destination).toBe(travelPlan1Name);
  });

  it('should allow a new travel plan to be added', () => {
    const searchBundle: fhir.Bundle = {
      resourceType: 'Bundle',
      type: 'searchset',
    };
    resourceServiceSpy.searchResource.and.returnValue(
        Promise.resolve(searchBundle));

    component.adding = true;
    fixture.detectChanges();

    const form = fixture.debugElement.query(By.directive(TravelFormComponent));
    expect(form).toBeTruthy();
  });
});
