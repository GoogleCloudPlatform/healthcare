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

import * as moment from 'moment';
import {ISO_DATE} from '../app/constants';

export function createTestTravelQuestionnaireResponse(
    destination = 'Test dest', departureDate: moment.Moment = moment(),
    returnDate: moment.Moment = moment()): fhir.QuestionnaireResponse {
  return {
    author: {reference: `Patient/1`},
    item: [
      {answer: [{valueString: destination}], linkId: '1'}, {
        item: [
          {
            answer: [{valueDate: departureDate.format(ISO_DATE)}],
            linkId: '2.1'
          },
          {answer: [{valueDate: returnDate.format(ISO_DATE)}], linkId: '2.2'}
        ],
        linkId: '2'
      }
    ],
    questionnaire: {reference: 'Questionnaire/travel-questionnaire'},
    resourceType: 'QuestionnaireResponse',
    status: 'completed',
    subject: {reference: 'Patient/1'}
  };
}
