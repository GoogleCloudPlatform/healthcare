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
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ReactiveFormsModule} from '@angular/forms';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatLegacyFormFieldModule} from '@angular/material/form-field';
import {MatLegacyInputModule} from '@angular/material/input';
import {MatLegacySelectModule} from '@angular/material/select';
import {By} from '@angular/platform-browser';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import * as moment from 'moment';
import {BehaviorSubject} from 'rxjs';

import {createTestImmunization} from '../../../test/immunization';
import {resourceServiceSpyProvider} from '../../../test/resource-service-spy';
import {ISO_DATE} from '../../constants';
import {Immunization} from '../immunization';
import {ImmunizationFormComponent} from '../immunization-form/immunization-form.component';
import {ImmunizationService} from '../immunization.service';

import {ImmunizationListComponent} from './immunization-list.component';

describe('ImmunizationListComponent', () => {
  let component: ImmunizationListComponent;
  let fixture: ComponentFixture<ImmunizationListComponent>;

  const immunizations = new BehaviorSubject([] as Immunization[]);
  const immunizationServiceStub = {
    immunizations$: immunizations.asObservable(),
  };

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          imports: [
            NoopAnimationsModule,
            MatDatepickerModule,
            ReactiveFormsModule,
            MatLegacySelectModule,
            MatLegacyFormFieldModule,
            MatLegacyInputModule,
          ],
          declarations: [
            ImmunizationListComponent,
            ImmunizationFormComponent,
          ],
          providers: [
            resourceServiceSpyProvider(),
            {provide: ImmunizationService, useValue: immunizationServiceStub},
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ImmunizationListComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should retrieve the collection of immunizations', async () => {
    const listener = new BehaviorSubject([] as Immunization[]);
    component.dueImmunizations$.subscribe(listener);

    immunizations.next([new Immunization(createTestImmunization())]);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(listener.getValue().length).toBe(1);
  });

  it('should split off the update to date immunizations', async () => {
    const imm = createTestImmunization();
    imm.expirationDate = moment().add(2, 'years').format(ISO_DATE);

    const listener = new BehaviorSubject([] as Immunization[]);
    component.upToDateImmunizations$.subscribe(listener);

    immunizations.next([new Immunization(imm)]);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(listener.getValue().length).toBe(1);
  });

  it('should order immunizations by date', async () => {
    const imm1Name = 'imm-1';
    const imm1 = createTestImmunization(imm1Name);
    imm1.id = '1';
    imm1.date = moment().subtract(2, 'years').format(ISO_DATE);
    const imm2Name = 'imm-2';
    const imm2 = createTestImmunization(imm2Name);
    imm2.id = '2';
    imm2.date = moment().subtract(1, 'years').format(ISO_DATE);

    const listener = new BehaviorSubject([] as Immunization[]);
    component.dueImmunizations$.subscribe(listener);

    immunizations.next([new Immunization(imm1), new Immunization(imm2)]);

    fixture.detectChanges();
    await fixture.whenStable();

    const imms = listener.getValue();
    expect(imms[0].name).toBe(imm2Name);
    expect(imms[1].name).toBe(imm1Name);
  });

  it('should allow a new immunization to be added', () => {
    component.add();
    fixture.detectChanges();

    expect(component.newImmunization).toBe(true);
    const form =
        fixture.debugElement.query(By.directive(ImmunizationFormComponent));
    expect(form).toBeTruthy();
  });
});
