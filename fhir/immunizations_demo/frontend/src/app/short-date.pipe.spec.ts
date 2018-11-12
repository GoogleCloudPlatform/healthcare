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

import {ShortDatePipe} from './short-date.pipe';

describe('ShortDatePipe', () => {
  const DATE_STRING = '2006-01-02T13:04:05';

  it('create an instance', () => {
    const pipe = new ShortDatePipe();
    expect(pipe).toBeTruthy();
  });

  it('should format a date string', () => {
    const pipe = new ShortDatePipe();
    expect(pipe.transform(DATE_STRING)).toEqual('Jan 2, 2006');
  });

  it('should format a moment date', () => {
    const pipe = new ShortDatePipe();
    const date = moment(DATE_STRING, moment.ISO_8601);
    expect(pipe.transform(date)).toEqual('Jan 2, 2006');
  });

  it('should parse the date using a custom format', () => {
    const pipe = new ShortDatePipe();
    expect(pipe.transform('02-01-2006', ['DD-MM-YYYY'])).toEqual('Jan 2, 2006');
  });
});
