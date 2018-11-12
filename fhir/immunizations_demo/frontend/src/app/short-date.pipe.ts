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
import {Pipe, PipeTransform} from '@angular/core';
import {isString} from 'lodash';
import * as moment from 'moment';

@Pipe({name: 'shortDate'})
export class ShortDatePipe implements PipeTransform {
  /**
   * Converts a date string or moment object into a short, locale-appropriate,
   * representation of a date.
   * @param value The date to display.
   * @param args The format used to parse value.
   */
  transform(value: string|moment.Moment, args?: string[]): string {
    if (isString(value)) {
      let format: moment.MomentFormatSpecification = moment.ISO_8601;
      if (args && args.length > 0) {
        format = args[0];
      }
      value = moment(value, format);
    }
    if (!(value instanceof moment)) {
      throw new Error('expected moment object or parseable date');
    }
    return value.format('ll');
  }
}
