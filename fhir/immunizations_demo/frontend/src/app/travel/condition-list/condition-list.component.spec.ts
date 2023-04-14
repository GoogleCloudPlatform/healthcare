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

// tslint:disable:no-any
import {NO_ERRORS_SCHEMA} from '@angular/core';
import {ComponentFixture, TestBed, waitForAsync} from '@angular/core/testing';
import {MatLegacySnackBar} from '@angular/material/legacy-snack-bar';
import {set} from 'lodash';
import {of} from 'rxjs';

import {createTestTravelQuestionnaireResponse} from '../../../test/travel-plan';
import {createTestScheduler} from '../../../test/util';
import {PredictionService} from '../../prediction.service';
import {TravelPlan} from '../travel-plan';

import {ConditionListComponent, RiskProbability} from './condition-list.component';

describe('ConditionListComponent', () => {
  const RISK_PROBABILITY_PATH = 'prediction[0].qualitativeRisk.coding[0].code';

  let component: ConditionListComponent;
  let fixture: ComponentFixture<ConditionListComponent>;
  const scheduler = createTestScheduler();
  const predictionServiceSpy = {
    predictions$: of([] as fhir.RiskAssessment[]),
  };
  const snackbarSpy =
      jasmine.createSpyObj<MatLegacySnackBar>('MatSnackBar', ['open']);

  beforeEach(waitForAsync(() => {
    TestBed
        .configureTestingModule({
          schemas: [NO_ERRORS_SCHEMA],
          declarations: [ConditionListComponent],
          providers: [
            {provide: PredictionService, useValue: predictionServiceSpy},
            {provide: MatLegacySnackBar, useValue: snackbarSpy},
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ConditionListComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should find the relevant risk assessments', () => {
    const fhirTravelPlan = createTestTravelQuestionnaireResponse();
    fhirTravelPlan.id = '1';
    component.travelPlan = new TravelPlan(fhirTravelPlan);

    const riskAssessment1: fhir.RiskAssessment = {
      id: 'RA1',
      basis: [
        {reference: `QuestionnaireResponse/${fhirTravelPlan.id}`},
      ],
      prediction: [{
        outcome: {coding: [{display: 'Yellow fever'}]},
        qualitativeRisk: {
          coding:
              [{code: 'low', system: 'http://hl7.org/fhir/risk-probability'}]
        }
      }]
    } as any;
    // Risk assessment for another trip
    const riskAssessment2: fhir.RiskAssessment = {
      id: 'RA2',
      basis: [
        {reference: 'QuestionnaireResponse/2'},
      ],
      prediction: [{
        outcome: {coding: [{display: 'Yellow fever'}]},
        qualitativeRisk: {
          coding:
              [{code: 'high', system: 'http://hl7.org/fhir/risk-probability'}]
        }
      }]
    } as any;

    scheduler.run(({cold, expectObservable}) => {
      const predictionService = TestBed.get(PredictionService);
      predictionService.predictions$ =
          cold('-r|', {r: [riskAssessment1, riskAssessment2]});

      component.ngOnInit();

      expectObservable(component.conditionAssessments$)
          .toBe('-l|', {e: [], l: [riskAssessment1]});
    });
  });

  it('should have the right icon classes', () => {
    const a: fhir.RiskAssessment = {} as any;

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Certain);
    expect(component.getIcon(a)).toEqual('report');

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Moderate);
    expect(component.getIcon(a)).toEqual('warning');

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Low);
    expect(component.getIcon(a)).toEqual('info');
  });

  it('should have the right icon colors', () => {
    const a: fhir.RiskAssessment = {} as any;

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Certain);
    expect(component.getIconColor(a)).toEqual('red');

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Moderate);
    expect(component.getIconColor(a)).toEqual('yellow');

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Low);
    expect(component.getIconColor(a)).toEqual('blue');
  });

  it('should display the correct severity', () => {
    const a: fhir.RiskAssessment = {} as any;

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Certain);
    expect(component.showOutcomeSeverity(a)).toEqual('High');

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Moderate);
    expect(component.showOutcomeSeverity(a)).toEqual('Moderate');

    set(a, RISK_PROBABILITY_PATH, RiskProbability.Low);
    expect(component.showOutcomeSeverity(a)).toEqual('Low');
  });

  it('should retrieve the predicted condition', () => {
    const a: fhir.RiskAssessment = {
      prediction: [
        {
          outcome: {
            coding: [{display: 'Malaria'}],
          },
        },
      ],
    } as any;
    expect(component.getPredictedCondition(a)).toEqual('Malaria');
  });
});
