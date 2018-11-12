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
import {isUndefined} from 'lodash';
import {TestScheduler} from 'rxjs/testing';

/**
 * Creates a test scheduler that uses `expect` to compare observable values.
 */
export function createTestScheduler(): TestScheduler {
  return new TestScheduler((actual, expected) => {
    expect(expected).toEqual(actual);
  });
}

/**
 * Resets all of the method calls on the given spy object.
 * @param spyObj The jasmine spy object to reset.
 */
export function resetSpyObj(
    spyObj: jasmine.SpyObj<{[spyMethod: string]: Function}>) {
  for (const m of Object.keys(spyObj)) {
    spyObj[m].calls.reset();
  }
}

/**
 * Asserts that `spy` throws an exception matching message when executed. This
 * is similar to the behaviour of expect(spy).toThrowError(message) except that
 * this function can be awaited, allowing for additional expectations to be run
 * after spy completes.
 * @param spy A function that executes the async method under test.
 * @param message An optional message that the error thrown by spy should match.
 */
export async function expectAsyncToThrowError(
    // tslint:disable-next-line:no-any
    spy: () => Promise<any>, message?: string|RegExp): Promise<void> {
  try {
    await spy();
    // An exception wasn't thrown, this is a failure.
    if (isUndefined(message)) {
      fail(`Expected error to be thrown`);
    } else {
      fail(`Expected error matching ${message} to be thrown`);
    }
  } catch (e) {
    if (!isUndefined(message)) {
      expect(e).toMatch(message);
    }
  }
}
