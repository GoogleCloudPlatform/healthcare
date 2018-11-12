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
import {ISO_DATE} from '../constants';
import {TravelPlan} from './travel-plan';

describe('TravelPlan', () => {
  const DESTINATION = 'Travel destination';
  const DEPARTURE_DATE = '1970-01-01';
  const RETURN_DATE = '1970-01-02';
  let questionnaireResponse: fhir.QuestionnaireResponse;

  beforeEach(() => {
    questionnaireResponse = {
      author: {reference: 'Patient/1'},
      item: [
        {answer: [{valueString: DESTINATION}], linkId: '1'}, {
          item: [
            {answer: [{valueDate: DEPARTURE_DATE}], linkId: '2.1'},
            {answer: [{valueDate: RETURN_DATE}], linkId: '2.2'}
          ],
          linkId: '2'
        }
      ],
      questionnaire: {reference: 'Questionnaire/1'},
      resourceType: 'QuestionnaireResponse',
      status: 'completed',
      subject: {reference: 'Patient/1'}
    };
  });

  it('should parse out the travel plan details', () => {
    const travelPlan = new TravelPlan(questionnaireResponse);

    expect(travelPlan.destination).toEqual(DESTINATION);
    expect(travelPlan.departureDate.isSame(moment(DEPARTURE_DATE, ISO_DATE)))
        .toBe(true);
    expect(travelPlan.returnDate.isSame(moment(RETURN_DATE, ISO_DATE)))
        .toBe(true);
  });

  it('should convert back to FHIR', () => {
    const travelPlan = new TravelPlan(questionnaireResponse);

    travelPlan.destination += ' 2';
    travelPlan.departureDate.add(1, 'days');
    travelPlan.returnDate.add(1, 'days');

    const fhirTravelPlan = travelPlan.toFHIR();

    expect(fhirTravelPlan.item![0].answer![0].valueString)
        .toEqual(DESTINATION + ' 2');
    expect(fhirTravelPlan.item![1].item![0].answer![0].valueDate)
        .toEqual('1970-01-02');
    expect(fhirTravelPlan.item![1].item![1].answer![0].valueDate)
        .toEqual('1970-01-03');
  });

  it('should throw errors if answers are missing', () => {
    delete questionnaireResponse.item![0].answer![0].valueString;
    delete questionnaireResponse.item![1].item![0].answer![0].valueDate;
    delete questionnaireResponse.item![1].item![1].answer![0].valueDate;

    expect(() => new TravelPlan(questionnaireResponse))
        .toThrowError('destination not found');

    questionnaireResponse.item![0].answer![0].valueString = DESTINATION;
    expect(() => new TravelPlan(questionnaireResponse))
        .toThrowError('departure date not found');

    questionnaireResponse.item![1].item![0].answer![0].valueDate =
        DEPARTURE_DATE;
    expect(() => new TravelPlan(questionnaireResponse))
        .toThrowError('return date not found');
  });
});
