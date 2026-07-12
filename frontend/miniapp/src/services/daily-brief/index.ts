import { MockDailyBriefAdapter, type MockDailyBriefScenario } from './mock-adapter';
import type { DailyBriefPort } from './port';

let scenario: MockDailyBriefScenario = 'ready';
let service: DailyBriefPort = new MockDailyBriefAdapter(scenario);

export function getDailyBriefService() {
  return service;
}

export function setDailyBriefMockScenario(nextScenario: MockDailyBriefScenario) {
  scenario = nextScenario;
  service = new MockDailyBriefAdapter(scenario);
}
