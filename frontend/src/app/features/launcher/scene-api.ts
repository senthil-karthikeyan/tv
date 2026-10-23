import { Injectable } from '@angular/core';
import {  httpResource } from '@angular/common/http';

export interface Scene {
id: string;
name: string;
label: string;
icon: string;
route: string;
order: number;
visible: boolean;
created_at: string;
updated_at: string;
}

interface ScenesResponse {
data: Scene[];
meta: { total: number };
}

@Injectable({ providedIn: 'root' })
export class SceneApi {

  scenes = httpResource<ScenesResponse>(() => '/api/v1/scenes');

}