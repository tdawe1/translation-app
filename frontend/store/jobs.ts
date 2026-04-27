/**
 * Jobs Store - Zustand state management for detected jobs
 */

import { create } from "zustand";

export interface Job {
  id: string;
  title: string;
  reward: number;
  url: string;
  source: string;
  currency?: string;
  timestamp?: string;
}

/** Extended job type with additional details from Gengo API */
export interface ExtendedJob extends Job {
  /** Optional full job description */
  description?: string;
  /** Source language code (e.g., "en", "ja") */
  sourceLanguage?: string;
  /** Target language code (e.g., "es", "fr") */
  targetLanguage?: string;
  /** Human-readable language pair (e.g., "English → Spanish") */
  languagePair?: string;
  /** Reason why job was filtered (if applicable) */
  filterReason?: string;
  /** Unit count for translation (words/characters) */
  unitCount?: number;
  /** Unit type: "words" or "characters" */
  unitType?: "words" | "characters";
  /** Deadline for job completion */
  deadline?: string;
  /** Whether job can be auto-accepted */
  canAccept?: boolean;
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
      const existingJobs = state.jobs.filter((existing) => existing.id !== job.id);
      const newJobs = [job, ...existingJobs].slice(0, state.maxJobs);
      return { jobs: newJobs };
    }),

  clearJobs: () => set({ jobs: [] }),

  removeJob: (id) =>
    set((state) => ({
      jobs: state.jobs.filter((job) => job.id !== id),
    })),
}));
