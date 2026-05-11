import { beforeEach, describe, expect, it } from 'vitest';

import { useRealtimeStore, type RealtimeEvent } from '@/store/realtime';

function makeEvent(id: string): RealtimeEvent {
  return {
    id,
    type: 'job.detected',
    message: `Event ${id}`,
    timestamp: `2026-04-23T00:00:0${id}Z`,
  };
}

describe('realtime store', () => {
  beforeEach(() => {
    useRealtimeStore.setState({
      events: [],
      maxEvents: 50,
      stats: {
        jobsDetected: 0,
        jobsAccepted: 0,
        jobsFiltered: 0,
        lastJobDetected: null,
      },
    });
  });

  it('respects maxEvents when hydrating an initial event batch', () => {
    useRealtimeStore.setState({ maxEvents: 2 });

    useRealtimeStore.getState().setEvents([makeEvent('1'), makeEvent('2'), makeEvent('3')]);

    const state = useRealtimeStore.getState();
    expect(state.events).toHaveLength(2);
    expect(state.events.map((event) => event.id)).toEqual(['1', '2']);
    expect(state.stats.jobsDetected).toBe(2);
  });

  it('dedupes hydrated events by id before rendering', () => {
    useRealtimeStore.getState().setEvents([makeEvent('1'), makeEvent('1'), makeEvent('2')]);

    const state = useRealtimeStore.getState();
    expect(state.events.map((event) => event.id)).toEqual(['1', '2']);
    expect(state.stats.jobsDetected).toBe(2);
  });
});
