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

/* A file that contains unit tests for the functions used in the background script. */

import {instanceFunctions} from '../helpers/background';
import 'jasmine';

const instance = [
  {
    id: '4',
    name: 'test-instance',
    status: 'RUNNING',
    description: '',
    zone: 'us-west1-b',
    disks: [
      {
        guestOsFeatures: [{}],
      },
    ],
    NetworkInterfaces: [
      {
        name: 'nic0',
        network: 'testNetwork',
        networkIP: 'ip',
      },
    ],
  },
];

// Test to make sure windows instances are initialized properly.
describe('background tests - creating windows instances', () => {
  it('windows instances are created properly', async () => {
    const windowsInstance = instance.slice();
    windowsInstance[0].disks[0].guestOsFeatures[0] = {type: 'WINDOWS'};
    spyOn(instanceFunctions, 'getInstancesApi').and.callFake(() => {
      return Promise.resolve(windowsInstance);
    });

    const testInstances = await instanceFunctions.getComputeInstances('test');
    expect(testInstances[0].name).toEqual('test-instance');
    expect(testInstances[0].displayPrivateRdpDom).toEqual(true);
  });
});

// Test to make sure non-windows instances are initialized properly.
describe('background tests - creating linux instances', () => {
  it('other instances are created properly', async () => {
    const linuxInstance = instance.slice();
    linuxInstance[0].disks[0].guestOsFeatures[0] = {type: 'LINUX'};
    spyOn(instanceFunctions, 'getInstancesApi').and.callFake(() => {
      return Promise.resolve(linuxInstance);
    });

    const testInstances = await instanceFunctions.getComputeInstances('test');
    expect(testInstances[0].name).toEqual('test-instance');
    expect(testInstances[0].displayPrivateRdpDom).toEqual(false);
  });
});
