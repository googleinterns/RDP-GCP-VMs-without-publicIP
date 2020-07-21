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

import { Component } from '@angular/core';
import { bindCallback, fromEventPattern, bindNodeCallback } from 'rxjs';
import { map } from 'rxjs/operators';
import { PopupService } from './popup.service';
import { Instance } from '../../../../../classes';
import { popupGetInstances, startPrivateRdp } from 'src/constants';

@Component({
  selector: 'app-popup',
  templateUrl: 'popup.component.html',
  providers: [PopupService],
  styleUrls: ['popup.component.scss']
})


export class PopupComponent {
  status: string;
  instances: Instance[];
  loading: boolean;
  projectName: string;

  constructor(private popupService: PopupService) {}

  ngOnInit() {
    this.loading = true;
    this.getStatus();
    this.getInstances();
  };

  getStatus() {
    this.popupService.getStatus()
      .subscribe((response: any) => {
        this.status = response.status;
      }, error => {
        this.status = 'Could not get server status';
      })
  };

  getInstances() {
    const pollForInstances = setInterval(() => {
        chrome.runtime.sendMessage({type: popupGetInstances}, (resp) => {
          if (resp.instances !== []) {
            this.loading = false;
            this.instances = resp.instances;
            this.projectName = resp.projectName;
          }
        });

        if (!this.loading) {
          clearInterval(pollForInstances);
        }
    }, 1000)
  };

  onRdpClick(instance: Instance) {
    chrome.runtime.sendMessage({type: startPrivateRdp, instance}, (resp) => {
      this.instances = resp.instances;
    });
  };

}
