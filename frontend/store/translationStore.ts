import { create } from "zustand";
import { toast } from "./toast";
import { translationApi } from "@/lib/api/translation";
import type {
  TranslationJob,
  TranslationSegment,
  CreateJobRequest,
  UpdateSegmentRequest,
  JobSummary,
  TranslationJobStatus,
} from "@/lib/api/types";


interface TranslationStoreState {
  jobs: JobSummary[];
  jobsLoading: boolean;
  jobsError: string | null;
  jobsPage: number;
  jobsPageSize: number;
  jobsTotalCount: number;
  jobsStatusFilter: TranslationJobStatus | "all";
  jobsSort: "newest" | "oldest";
  currentJob: TranslationJob | null;
  currentJobLoading: boolean;
  currentJobError: string | null;
  flaggedSegments: TranslationSegment[];
  flaggedSegmentsLoading: boolean;
  flaggedSegmentsError: string | null;
  fetchJobs: (params?: {
    page?: number;
    pageSize?: number;
    status?: TranslationJobStatus | "all";
    sort?: "newest" | "oldest";
  }) => Promise<void>;
  fetchJob: (jobId: string) => Promise<void>;
  createJob: (data: CreateJobRequest) => Promise<TranslationJob>;
  approveJob: (jobId: string) => Promise<void>;
  rejectJob: (jobId: string) => Promise<void>;
  updateSegment: (jobId: string, segmentUuid: string, data: UpdateSegmentRequest) => Promise<void>;
  fetchFlaggedSegments: (jobId: string) => Promise<void>;
  clear: () => void;
  setCurrentJob: (job: TranslationJob | null) => void;
}

export const useTranslationStore = create<TranslationStoreState>()((set, get) => ({
  jobs: [],
  jobsLoading: false,
  jobsError: null,
  jobsPage: 1,
  jobsPageSize: 10,
  jobsTotalCount: 0,
  jobsStatusFilter: "all",
  jobsSort: "newest",
  currentJob: null,
  currentJobLoading: false,
  currentJobError: null,
  flaggedSegments: [],
  flaggedSegmentsLoading: false,
  flaggedSegmentsError: null,

  fetchJobs: async (params) => {
    set({ jobsLoading: true, jobsError: null });
    const state = get();
    const page = params?.page ?? state.jobsPage;
    const pageSize = params?.pageSize ?? state.jobsPageSize;
    const status = params?.status ?? state.jobsStatusFilter;
    const sort = params?.sort ?? state.jobsSort;

    try {
      const response = await translationApi.listJobs({
        page,
        page_size: pageSize,
        status: status === "all" ? undefined : status,
        sort,
      });
      set({
        jobs: response.jobs,
        jobsPage: response.page,
        jobsPageSize: response.page_size,
        jobsTotalCount: response.total_count,
        jobsStatusFilter: status,
        jobsSort: sort,
        jobsLoading: false,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to fetch jobs";
      set({ jobsError: message, jobsLoading: false });
      toast.error(message);
    }
  },

  fetchJob: async (jobId: string) => {
    set({ currentJobLoading: true, currentJobError: null });
    try {
      const job = await translationApi.getJob(jobId);
      set({ currentJob: job, currentJobLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to fetch job";
      set({ currentJobError: message, currentJobLoading: false });
      toast.error(message);
    }
  },

  createJob: async (data: CreateJobRequest) => {
    set({ jobsLoading: true, jobsError: null });
    try {
      const job = await translationApi.createJob(data);
      const state = get();
      await state.fetchJobs({
        page: 1,
        pageSize: state.jobsPageSize,
        status: state.jobsStatusFilter,
        sort: state.jobsSort,
      });
      toast.success("Job created successfully");
      return job;
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to create job";
      set({ jobsError: message, jobsLoading: false });
      toast.error(message);
      throw error;
    }
  },

  approveJob: async (jobId: string) => {
    set({ jobsLoading: true, jobsError: null });
    try {
      await translationApi.approveJob(jobId);
      set((state) => ({
        jobs: state.jobs.map((job) =>
          job.id === jobId ? { ...job, status: "approved" } : job
        ),
        currentJob: state.currentJob?.id === jobId
          ? { ...state.currentJob, status: "approved" }
          : state.currentJob,
        jobsLoading: false,
      }));
      toast.success("Job approved");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to approve job";
      set({ jobsError: message, jobsLoading: false });
      toast.error(message);
      throw error;
    }
  },

  rejectJob: async (jobId: string) => {
    set({ jobsLoading: true, jobsError: null });
    try {
      await translationApi.rejectJob(jobId);
      set((state) => ({
        jobs: state.jobs.map((job) =>
          job.id === jobId ? { ...job, status: "rejected" } : job
        ),
        currentJob: state.currentJob?.id === jobId
          ? { ...state.currentJob, status: "rejected" }
          : state.currentJob,
        jobsLoading: false,
      }));
      toast.success("Job rejected");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to reject job";
      set({ jobsError: message, jobsLoading: false });
      toast.error(message);
      throw error;
    }
  },

  updateSegment: async (jobId: string, segmentUuid: string, data: UpdateSegmentRequest) => {
    set({ jobsLoading: true, jobsError: null });
    try {
      const updatedSegment = await translationApi.updateSegment(jobId, segmentUuid, data);

      set((state) => ({
        currentJob: state.currentJob?.id === jobId
          ? {
              ...state.currentJob,
              segments: state.currentJob.segments?.map((seg: TranslationSegment) =>
                seg.id === segmentUuid ? updatedSegment : seg
              ),
            }
          : state.currentJob,
        flaggedSegments: state.flaggedSegments.map((seg) =>
          seg.id === segmentUuid ? updatedSegment : seg
        ),
        jobsLoading: false,
      }));
      toast.success("Segment updated");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to update segment";
      set({ jobsError: message, jobsLoading: false });
      toast.error(message);
      throw error;
    }
  },

  fetchFlaggedSegments: async (jobId: string) => {
    set({ flaggedSegmentsLoading: true, flaggedSegmentsError: null });
    try {
      const response = await translationApi.getFlaggedSegments(jobId);
      set({
        flaggedSegments: response.segments,
        flaggedSegmentsLoading: false,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to fetch flagged segments";
      set({ flaggedSegmentsError: message, flaggedSegmentsLoading: false });
      toast.error(message);
    }
  },

  setCurrentJob: (job) => set({ currentJob: job }),

  clear: () =>
    set({
      jobs: [],
      jobsError: null,
      jobsLoading: false,
      jobsPage: 1,
      jobsPageSize: 10,
      jobsTotalCount: 0,
      jobsStatusFilter: "all",
      jobsSort: "newest",
      currentJob: null,
      currentJobError: null,
      currentJobLoading: false,
      flaggedSegments: [],
      flaggedSegmentsError: null,
      flaggedSegmentsLoading: false,
    }),
}));
