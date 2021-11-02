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

import {Overlay, OverlayRef} from '@angular/cdk/overlay';
import {CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ComponentFixture, TestBed, waitForAsync} from '@angular/core/testing';

import {createTestScheduler, resetSpyObj} from '../../test/util';
import {PatientService} from '../patient.service';

import {PatientPanelComponent} from './patient-panel.component';

describe('PatientPanelComponent', () => {
  let component: PatientPanelComponent;
  let fixture: ComponentFixture<PatientPanelComponent>;
  let patientData: fhir.Patient;
  const overlayRefSpy = jasmine.createSpyObj<OverlayRef>(
      'OverlayRef', ['attach', 'detach', 'backdropClick']);
  // tslint:disable-next-line:no-any
  const fluentAPIMock: any = new Proxy(() => fluentAPIMock, {
    get() {
      return fluentAPIMock;
    },
  });
  const overlaySpy = jasmine.createSpyObj<Overlay>('Overlay', {
    create: overlayRefSpy,
    position: fluentAPIMock,
  });
  const scheduler = createTestScheduler();
  const patientServiceSpy =
      jasmine.createSpyObj('PatientService', ['getPatient']);

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [CUSTOM_ELEMENTS_SCHEMA],
          declarations: [PatientPanelComponent],
          providers: [
            {provide: Overlay, useValue: overlaySpy},
            {provide: PatientService, useValue: patientServiceSpy},
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    patientData = {
      birthDate: '1970-01-01',
      name: [{given: ['Test'], family: 'Patient'}],
      telecom: [
        {value: '(555) 555-5555'},
      ],
      gender: 'other',
    };
    resetSpyObj(patientServiceSpy);

    spyOn(Date, 'now').and.callFake(() => new Date('2000-01-02T00:00:00.000'));

    fixture = TestBed.createComponent(PatientPanelComponent);
    component = fixture.componentInstance;
  });

  it('should populate the patient data', () => {
    scheduler.run(({cold}) => {
      patientServiceSpy.getPatient.and.returnValue(
          cold('p|', {p: patientData}));

      fixture.detectChanges();
    });

    expect(component.patient.age).toEqual(30);
    expect(component.patient.name).toEqual('Patient, Test');
    expect(component.patient.phoneNumber).toEqual('(555) 555-5555');
    expect(component.patient.gender).toEqual('Other');
  });

  it('should accept missing patient data', () => {
    scheduler.run(({cold}) => {
      patientServiceSpy.getPatient.and.returnValue(cold('p|', {p: {}}));

      fixture.detectChanges();
    });

    expect(component.patient.age).toEqual(undefined);
    expect(component.patient.name).toEqual('');
    expect(component.patient.phoneNumber).toEqual('');
    expect(component.patient.gender).toEqual('');
  });

  it('should open and close the overlay', () => {
    fixture.detectChanges();

    scheduler.run(({flush, hot}) => {
      scheduler.schedule(() => component.toggle(), 1);
      overlayRefSpy.backdropClick.and.returnValue(hot('-'));
      flush();

      expect(overlayRefSpy.attach).toHaveBeenCalled();

      scheduler.schedule(() => component.toggle(), 1);
      flush();

      expect(overlayRefSpy.detach).toHaveBeenCalled();
    });
  });

  it('should close the overlay if the backdrop is clicked', () => {
    fixture.detectChanges();

    scheduler.run(({hot}) => {
      scheduler.schedule(() => component.toggle(), 1);
      overlayRefSpy.backdropClick.and.returnValue(hot('-e', {e: undefined}));
    });

    expect(overlayRefSpy.detach).toHaveBeenCalled();
  });
});
