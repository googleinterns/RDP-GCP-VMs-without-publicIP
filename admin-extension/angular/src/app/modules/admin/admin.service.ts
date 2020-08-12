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

import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { verifyTokenEndpoint, getConfigEndpoint, getComputeInstancesEndpoint, sendOperationEndpoint, sendProjectOperationEndpoint, runPreRDPOperationsEndpoint } from 'src/constants';

const httpOptions = {
    headers: new HttpHeaders({ 'Content-Type': 'application/json' }),
    withCredentials: true, //this is required so that Angular returns the Cookies received from the server. The server sends cookies in Set-Cookie header. Without this, Angular will ignore the Set-Cookie header
};

@Injectable()
export class AdminService {
    constructor(private http: HttpClient){}

    getConfig (): Observable<object> {
        return this.http.get(getConfigEndpoint, httpOptions)
    }

    verifyToken (data: object): Observable<object> {
        return this.http.post(verifyTokenEndpoint, data, httpOptions);
    }

    getComputeInstances (data: object): Observable<object> {
        return this.http.post(getComputeInstancesEndpoint, data, httpOptions)
    }

    sendOperation (data: object): Observable<object> {
        return this.http.post(sendOperationEndpoint, data, httpOptions)
    }

    sendProjectOperation(data: object): Observable<object> {
        return this.http.post(sendProjectOperationEndpoint, data, httpOptions)
    }

    runPreRDPOperations(data: object): Observable<object> {
        return this.http.post(runPreRDPOperationsEndpoint, data, httpOptions)
    }
}
