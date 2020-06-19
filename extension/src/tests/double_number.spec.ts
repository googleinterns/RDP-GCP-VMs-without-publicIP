/***
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
***/

/* A file that contains a test for a function that doubles the number passed in.
Used solely for testing purposes and will be removed in the future. */

import { doubleNumber } from "../double_number";
describe("doubleNumber tests", () => {
    it("doubles 2 to equal 4", () => {
        expect(doubleNumber(2)).toBe(4);
    })
})