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
import * as moment from 'moment';

import {createTestImmunization, createTestReaction} from '../../../test/immunization';
import {createResourceServiceSpy, resourceServiceSpyProvider} from '../../../test/resource-service-spy';
import {resetSpyObj} from '../../../test/util';
import {Immunization, ImmunizationReaction} from '../immunization';
import {ImmunizationFormComponent} from '../immunization-form/immunization-form.component';
import {ImmunizationService} from '../immunization.service';

import {ImmunizationItemComponent} from './immunization-item.component';

describe('ImmunizationItemComponent', () => {
  let component: ImmunizationItemComponent;
  let fixture: ComponentFixture<ImmunizationItemComponent>;
  let fhirImm: fhir.Immunization;

  const resourceServiceSpy = createResourceServiceSpy();
  const immunizationServiceSpy: jasmine.SpyObj<ImmunizationService> =
      jasmine.createSpyObj(
          'ImmunizationService', ['create', 'save', 'remove', 'addNewDose']);

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          declarations: [ImmunizationItemComponent],
          providers: [
            resourceServiceSpyProvider(resourceServiceSpy),
            {provide: ImmunizationService, useValue: immunizationServiceSpy},
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ImmunizationItemComponent);
    component = fixture.componentInstance;

    fhirImm = createTestImmunization();
    fhirImm.id = '0';
    component.immunization = new Immunization(fhirImm);

    resetSpyObj(resourceServiceSpy);
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should set the header to expiring soon', () => {
    const expiry = moment().add(1, 'month');
    component.immunization.expirationDate = expiry;

    fixture.detectChanges();

    expect(component.header).toEqual(`Due ${expiry.fromNow()}`);
  });

  it('should set the header to received date', () => {
    component.immunization.expirationDate = moment().add(1, 'year');

    fixture.detectChanges();

    const received = component.immunization.date;
    expect(component.header).toEqual(`Received on ${received.format('ll')}`);
  });

  it('should return the reaction', async () => {
    const reactionSymptom = ImmunizationReaction.Fever;
    const reaction = createTestReaction('0-reaction', reactionSymptom);
    component.immunization.reactionID = reaction.id;
    resourceServiceSpy.getResource.and.returnValue(Promise.resolve(reaction));

    fixture.detectChanges();
    const fetchedReaction = await component.getReaction();

    expect(fetchedReaction).toBe(reaction);
    expect(component.reactionDetails).toEqual(reactionSymptom);
  });

  it('should noop if there is no reaction', async () => {
    fixture.detectChanges();
    const fetchedReaction = await component.getReaction();

    expect(fetchedReaction).toBeUndefined();
    expect(component.reactionDetails).toBeUndefined();
    expect(resourceServiceSpy.getResource).not.toHaveBeenCalled();
  });

  it('should fetch the reaction once', async () => {
    const reaction = createTestReaction();
    component.immunization.reactionID = reaction.id;
    resourceServiceSpy.getResource.and.returnValue(Promise.resolve(reaction));

    fixture.detectChanges();
    await component.getReaction();
    await component.getReaction();

    expect(resourceServiceSpy.getResource).toHaveBeenCalledTimes(1);
  });

  it('should save the immunization', async () => {
    const immFormSpy = jasmine.createSpyObj<ImmunizationFormComponent>(
        'ImmunizationFormComponent', ['submit']);
    component.immForm = immFormSpy;
    const reaction = createTestReaction();
    resourceServiceSpy.getResource.and.returnValue(Promise.resolve(reaction));

    fixture.detectChanges();
    component.immunization.reactionID = reaction.id;
    const savePromise = component.save();
    expect(component.loading).toBe(true);
    await savePromise;
    expect(component.loading).toBe(false);

    expect(immFormSpy.submit).toHaveBeenCalled();
    expect(resourceServiceSpy.getResource).toHaveBeenCalled();
  });

  it('delete the immunization and reaction', async () => {
    const reaction = createTestReaction();
    component.immunization.reactionID = reaction.id;
    fhirImm.id = '0';
    resourceServiceSpy.getResource.and.returnValue(Promise.resolve(reaction));

    fixture.detectChanges();
    await component.getReaction();
    const deletePromise = component.delete();
    expect(component.loading).toBe(true);
    await deletePromise;
    expect(component.loading).toBe(false);

    expect(resourceServiceSpy.deleteResource)
        .toHaveBeenCalledWith(jasmine.objectContaining({
          resourceType: 'Observation',
          id: reaction.id,
        }));
    expect(immunizationServiceSpy.remove)
        .toHaveBeenCalledWith(component.immunization);
  });

  it('should mark the immunization as updated', async () => {
    const updatedPromise = component.markUpdated();
    expect(component.loading).toBe(true);
    await updatedPromise;
    expect(component.loading).toBe(false);

    expect(immunizationServiceSpy.save)
        .toHaveBeenCalledWith(component.immunization);
  });

  it('should update the immunization', async () => {
    const updatedPromise = component.updateImm();
    expect(component.loading).toBe(true);
    await updatedPromise;
    expect(component.loading).toBe(false);

    expect(immunizationServiceSpy.addNewDose)
        .toHaveBeenCalledWith(component.immunization);
  });
});
