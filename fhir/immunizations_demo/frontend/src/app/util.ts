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

/**
 * A template string function for escaping URL parts.
 * @param urlStrings The URL parts that do not require escaping.
 * @param unsafeComponents The URL parts the do require escaping.
 */
export function encodeURL(
    urlStrings: TemplateStringsArray, ...unsafeComponents: string[]): string {
  let i = 0;
  let j = 0;
  let result = '';
  while (i < urlStrings.length || j < unsafeComponents.length) {
    if (i < urlStrings.length) {
      result += urlStrings[i];
    }
    if (j < unsafeComponents.length) {
      result += encodeURIComponent(unsafeComponents[j]);
    }
    i += 1;
    j += 1;
  }
  return result;
}
