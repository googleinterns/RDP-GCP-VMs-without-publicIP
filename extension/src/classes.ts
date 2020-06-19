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

interface GuestOsFeature {
    type: string;
}

interface Disk {
    guestOsFeatures: GuestOsFeature[];
}

interface NetworkInterface {
    name: string;
    network: string;
    networkIP: string;
}

interface InstanceInterface {
    id: string;
    name: string;
    status: string;
    description: string;
    zone: string;
    disks: Disk[];
    NetworkInterfaces: NetworkInterface[];
}

class Instance implements InstanceInterface {
    constructor(instance: InstanceInterface) {
        this.name = instance.name;
        this.status = instance.status;
        this.id = instance.id;
        this.zone = instance.zone;
        this.disks = instance.disks;
        this.description = instance.description
        this.NetworkInterfaces = instance.NetworkInterfaces;
        this.setRdpDomDisplay();
    }

    setRdpDomDisplay() {
        for (let i = 0; i < this.disks.length; i++) {
            for (let j = 0; j < this.disks[i].guestOsFeatures.length; j++) {
                if (this.disks[i].guestOsFeatures[j].type === "WINDOWS") {
                    this.displayPrivateRdpDom = true;
                    return;
                }
            }
        }
    }

    NetworkInterfaces: NetworkInterface[];
    description: string;
    disks: Disk[];
    id: string;
    name: string;
    status: string;
    zone: string;
    displayPrivateRdpDom: boolean;
}

export { Instance, InstanceInterface };
