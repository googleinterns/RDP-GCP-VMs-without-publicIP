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
import { bindCallback } from 'rxjs';
import { map } from 'rxjs/operators';
import { PopupService } from './popup.service';

@Component({
  selector: 'app-popup',
  templateUrl: 'popup.component.html',
  providers: [PopupService],
  styleUrls: ['popup.component.scss']
})
export class PopupComponent {
  status: string;

  constructor(private popupService: PopupService) {}

  ngOnInit() {
    this.getStatus();
  };

  getStatus() {
    this.popupService.getStatus()
      .subscribe(response => {
        this.status = response['status'];
      }, error => {
        this.status = 'Could not get server status';
      })
  }
}
