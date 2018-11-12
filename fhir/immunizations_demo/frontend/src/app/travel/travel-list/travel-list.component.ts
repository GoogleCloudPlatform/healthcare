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
import {trigger} from '@angular/animations';
import {Component, OnInit} from '@angular/core';
import {orderBy, partition} from 'lodash';
import * as moment from 'moment';
import {OPEN_IN} from '../../animations';
import {ResourceService} from '../../resource.service';
import {TravelPlan} from '../travel-plan';

@Component({
  selector: 'app-travel-list',
  templateUrl: './travel-list.component.html',
  styleUrls: [
    './travel-list.component.scss',
  ],
  animations: [trigger('openIn', OPEN_IN)],
})
export class TravelListComponent implements OnInit {
  upcomingTravel: TravelPlan[] = [];
  pastTravel: TravelPlan[] = [];
  adding = false;
  questionnaire: fhir.Questionnaire|undefined;

  constructor(private readonly resourceService: ResourceService) {}

  ngOnInit(): void {
    this.refreshTravelPlans();
  }

  async refreshTravelPlans(): Promise<void> {
    /*
     * The travel plan form is modelled in FHIR with a questionnaire. Since all
     * travel plans use the same questionnaire it is given an
     * application-specific identifier. The _revinclude parameter tells the
     * server to return all QuestionnaireResponses that are linked to the
     * travel questionnaire via the "questionnaire" field, in addition to the
     * source questionnaire.
     */
    const bundle = await this.resourceService.searchResource(
        'Questionnaire',
        {
          identifier: 'travel-questionnaire',
          _revinclude: 'QuestionnaireResponse:questionnaire',
        },
    );

    const entries = bundle.entry || [];
    const [questionnaire, responses] = partition(
        entries.map(e => e.resource as fhir.Resource),
        r => r.resourceType === 'Questionnaire');
    if (questionnaire.length === 0) {
      return;
    }
    this.questionnaire = questionnaire[0] as fhir.Questionnaire;
    this.parseResponses(responses as fhir.QuestionnaireResponse[]);
  }

  savedTravelForm() {
    this.adding = false;
    this.refreshTravelPlans();
  }

  hideTravelForm() {
    this.adding = false;
  }

  showTravelForm() {
    this.adding = true;
  }

  private parseResponses(responses: fhir.QuestionnaireResponse[]): void {
    let travelPlans = responses.map(r => new TravelPlan(r));
    travelPlans = orderBy(travelPlans, ['departureDate'], ['desc']);

    const today = moment().startOf('day');
    [this.upcomingTravel, this.pastTravel] =
        partition(travelPlans, r => r.departureDate.isSameOrAfter(today));
  }
}
