/**
 * Jobs Store - Zustand state management for detected jobs
 */

import { create } from "zustand";

export interface Job {
  id: string;
  title: string;
  reward: number;
  url: string;
  source: "rss" | "websocket";
  timestamp?: string;
}

interface JobsStoreState {
  jobs: Job[];
  maxJobs: number; // Maximum jobs to keep in memory

  // Actions
  addJob: (job: Job) => void;
  clearJobs: () => void;
  removeJob: (id: string) => void;
}

export const useJobsStore = create<JobsStoreState>()((set, get) => ({
  jobs: [],
  maxJobs: 100,

  addJob: (job) =>
    set((state) => {
      const newJobs = [job, ...state.jobs].slice(0, state.maxJobs);
      return { jobs: newJobs };
    }),

  clearJobs: () => set({ jobs: [] }),

  removeJob: (id) =>
    set((state) => ({
      jobs: state.jobs.filter((job) => job.id !== id),
    })),
}));
