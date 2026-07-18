import { describe, expect, it, vi } from 'vitest';
import { createAIService } from './ai-service';
import { createEventService } from './event-service';
import { createSectorService } from './sector-service';
import { createSubscriptionService } from './subscription-service';

describe('frontend mock adapter injection', () => {
  it('lets each mock-backed service use a replacement adapter', async () => {
    const loadEvents = vi.fn().mockResolvedValue([]);
    const loadSectors = vi.fn().mockResolvedValue([]);
    const loadSubscriptions = vi.fn().mockResolvedValue([]);
    const loadGreeting = vi.fn().mockResolvedValue({ id: 'custom' });

    await createEventService(loadEvents).getEventHighlights();
    await createSectorService(loadSectors).getSectorSignals();
    await createSubscriptionService(loadSubscriptions).getSubscriptionTopics();
    await createAIService(loadGreeting).getAssistantGreeting();

    expect(loadEvents).toHaveBeenCalledOnce();
    expect(loadSectors).toHaveBeenCalledOnce();
    expect(loadSubscriptions).toHaveBeenCalledOnce();
    expect(loadGreeting).toHaveBeenCalledOnce();
  });
});
