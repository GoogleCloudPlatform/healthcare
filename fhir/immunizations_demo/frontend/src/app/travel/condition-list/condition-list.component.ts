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
import {Component, Input, OnDestroy, OnInit} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';
import {get, includes, isUndefined, sortBy} from 'lodash';
import * as moment from 'moment';
import {Observable, of, Subject} from 'rxjs';
import {map, takeUntil} from 'rxjs/operators';

import {PredictionService} from '../../prediction.service';
import {TravelPlan} from '../travel-plan';

export enum RiskProbability {
  Negligible = 'negligible',
  Low = 'low',
  Moderate = 'moderate',
  High = 'high',
  Certain = 'certain',
}

enum RiskSeverity {
  High,
  Moderate,
  Low,
}

@Component({
  selector: 'app-condition-list',
  templateUrl: './condition-list.component.html',
  styles: [
    '.icon-red { color: #F4131E; }',
    '.icon-yellow { color: #F9BF13; }',
    '.icon-blue { color: #2137AA; }',
    'span.icon-red, span.icon-yellow, span.icon-blue { cursor: pointer; }',
  ]
})
export class ConditionListComponent implements OnInit, OnDestroy {
  @Input() travelPlan!: TravelPlan;
  conditionAssessments$: Observable<fhir.RiskAssessment[]>;

  private lastUpdated: Date|undefined;
  private travelPlanID = '';
  private destroyed$ = new Subject();

  constructor(
      private readonly predictionService: PredictionService,
      private readonly snackbar: MatSnackBar) {
    this.conditionAssessments$ = of([]);
  }

  ngOnInit(): void {
    const fhirTravelPlan = this.travelPlan.toFHIR();
    this.travelPlanID = `${fhirTravelPlan.resourceType}/${fhirTravelPlan.id}`;
    this.initAssessmentStream();
  }

  ngOnDestroy(): void {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  getIcon(a: fhir.RiskAssessment): string {
    const severity = this.getOutcomeSeverity(a);
    switch (severity) {
      case RiskSeverity.High:
        return 'report';
      case RiskSeverity.Moderate:
        return 'warning';
      case RiskSeverity.Low:
        return 'info';
      default:
        throw new Error(`Unexpected risk severity ${severity}`);
    }
  }

  getIconColor(a: fhir.RiskAssessment): string {
    const severity = this.getOutcomeSeverity(a);
    switch (severity) {
      case RiskSeverity.High:
        return 'red';
      case RiskSeverity.Moderate:
        return 'yellow';
      case RiskSeverity.Low:
        return 'blue';
      default:
        throw new Error(`Unexpected risk severity ${severity}`);
    }
  }

  getPredictedCondition(a: fhir.RiskAssessment): string {
    return get(a, 'prediction[0].outcome.coding[0].display');
  }

  showOutcomeSeverity(a: fhir.RiskAssessment): string {
    const severity = this.getOutcomeSeverity(a);
    switch (severity) {
      case RiskSeverity.High:
        return 'High';
      case RiskSeverity.Moderate:
        return 'Moderate';
      case RiskSeverity.Low:
        return 'Low';
      default:
        throw new Error(`Unexpected risk severity ${severity}`);
    }
  }

  private initAssessmentStream(): void {
    this.conditionAssessments$ = this.predictionService.predictions$.pipe(
        takeUntil(this.destroyed$),
        map(predictions => {
          const assessments = predictions.filter(p => {
            const basis = (p.basis || []);
            return basis.some(
                ref => includes(ref.reference, this.travelPlanID));
          });
          if (this.lastUpdated) {
            const hasUpdates = this.hasRecentUpdate(assessments);
            if (hasUpdates) {
              this.snackbar.open(
                  'Condition risks have been updated.',
                  undefined,
                  {
                    duration: 2000,
                  },
              );
            }
          }
          this.lastUpdated = new Date();
          return sortBy(
              assessments,
              [(a: fhir.RiskAssessment) => this.getOutcomeSeverity(a)]);
        }),
    );
  }

  private getOutcomeSeverity(a: fhir.RiskAssessment): RiskSeverity {
    const probability: RiskProbability =
        get(a, 'prediction[0].qualitativeRisk.coding[0].code');
    switch (probability) {
      case RiskProbability.Certain:
      case RiskProbability.High:
        return RiskSeverity.High;
      case RiskProbability.Moderate:
        return RiskSeverity.Moderate;
      case RiskProbability.Low:
      case RiskProbability.Negligible:
        return RiskSeverity.Low;
      default:
        throw new Error(`Unexpected risk probability ${probability}`);
    }
  }

  private hasRecentUpdate(assessments: fhir.RiskAssessment[]): boolean {
    return assessments.some(
        a => !isUndefined(a.meta) &&
            moment(a.meta.lastUpdated, moment.ISO_8601)
                .isAfter(this.lastUpdated));
  }
}
