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


import {get, isUndefined, set} from 'lodash';
import * as moment from 'moment';
import {ISO_DATE} from '../constants';

const QUESTIONNAIRE_PATHS = Object.freeze({
  destination: 'item[0].answer[0].valueString',
  departureDate: 'item[1].item[0].answer[0].valueDate',
  returnDate: 'item[1].item[1].answer[0].valueDate',
});

/**
 * Represents a user's intention to visit a destination, either in the past or
 * future.
 */
export class TravelPlan {
  /**
   * The destination of this trip.
   */
  destination: string;

  /**
   * The date of departure.
   */
  departureDate: moment.Moment;

  /**
   * The date of return to the source country.
   */
  returnDate: moment.Moment;

  /**
   * Parse a response to the travel questionnaire into a format that is easier
   * to use.
   * @param questionnaireResponse.
   */
  constructor(private questionnaireResponse: fhir.QuestionnaireResponse) {
    this.destination =
        get(questionnaireResponse, QUESTIONNAIRE_PATHS.destination);
    if (isUndefined(this.destination)) {
      throw new Error('destination not found');
    }

    const departureDate =
        get(questionnaireResponse, QUESTIONNAIRE_PATHS.departureDate);
    if (isUndefined(departureDate)) {
      throw new Error('departure date not found');
    }

    const returnDate =
        get(questionnaireResponse, QUESTIONNAIRE_PATHS.returnDate);
    if (isUndefined(returnDate)) {
      throw new Error('return date not found');
    }

    this.departureDate = moment(departureDate, ISO_DATE);
    this.returnDate = moment(returnDate, ISO_DATE);
  }

  toFHIR(): fhir.QuestionnaireResponse {
    set(this.questionnaireResponse, QUESTIONNAIRE_PATHS.destination,
        this.destination);
    set(this.questionnaireResponse, QUESTIONNAIRE_PATHS.departureDate,
        this.departureDate.format(ISO_DATE));
    set(this.questionnaireResponse, QUESTIONNAIRE_PATHS.returnDate,
        this.returnDate.format(ISO_DATE));

    return this.questionnaireResponse;
  }
}
