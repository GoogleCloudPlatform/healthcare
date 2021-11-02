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
import {MatAutocompleteModule} from '@angular/material/autocomplete';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {By} from '@angular/platform-browser';

import {createResourceServiceSpy, resourceServiceSpyProvider} from '../../../test/resource-service-spy';
import {createTestTravelQuestionnaireResponse} from '../../../test/travel-plan';
import {ShortDatePipe} from '../../short-date.pipe';
import {TravelFormComponent} from '../travel-form/travel-form.component';
import {TravelPlan} from '../travel-plan';

import {TravelItemComponent} from './travel-item.component';

describe('TravelItemComponent', () => {
  let component: TravelItemComponent;
  let fixture: ComponentFixture<TravelItemComponent>;

  const resourceServiceSpy = createResourceServiceSpy();
  let travelPlan: fhir.QuestionnaireResponse;

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          imports: [
            ReactiveFormsModule,
            MatDatepickerModule,
            MatAutocompleteModule,
          ],
          declarations: [
            TravelItemComponent,
            TravelFormComponent,
            ShortDatePipe,
          ],
          providers: [resourceServiceSpyProvider(resourceServiceSpy)],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(TravelItemComponent);
    component = fixture.componentInstance;
    travelPlan = createTestTravelQuestionnaireResponse();
    travelPlan.id = '1';
    component.travelPlan = new TravelPlan(travelPlan);
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should save the form', async () => {
    fixture.detectChanges();

    component.editing = true;
    fixture.detectChanges();

    const travelFormElement =
        fixture.debugElement.query(By.directive(TravelFormComponent));
    const travelForm = travelFormElement.componentInstance;
    const submitSpy = spyOn(travelForm, 'submit');
    const savePromise = component.save();
    expect(component.loading).toBeTruthy();
    await savePromise;

    expect(submitSpy).toHaveBeenCalled();
    expect(component.editing).toBeFalsy();
    expect(component.loading).toBeFalsy();
  });

  it('should cancel editing', async () => {
    fixture.detectChanges();

    component.editing = true;
    fixture.detectChanges();

    const travelFormElement =
        fixture.debugElement.query(By.directive(TravelFormComponent));
    const travelForm = travelFormElement.componentInstance;
    const cancelSpy = spyOn(travelForm, 'cancel');

    component.cancel();
    fixture.detectChanges();

    expect(cancelSpy).toHaveBeenCalled();
    expect(component.editing).toBeFalsy();
  });

  it('should delete the travel plan', async () => {
    const deletePromise = component.delete();
    expect(component.loading).toBeTruthy();
    await deletePromise;
    expect(component.loading).toBeFalsy();

    expect(resourceServiceSpy.deleteResource)
        .toHaveBeenCalledWith(jasmine.objectContaining({
          resourceType: 'QuestionnaireResponse',
          id: travelPlan.id,
        }));
  });
});
