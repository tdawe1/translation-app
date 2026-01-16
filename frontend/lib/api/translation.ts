import { client } from "./client";
import type {
  TranslationJob,
  TranslationSegment,
  ListJobsResponse,
  CreateJobRequest,
  UpdateSegmentRequest,
  RejectJobRequest,
  FlaggedSegmentsResponse,
  TranslationJobStatus,
} from "./types";

export const translationApi = {
  listJobs: (params?: {
    status?: TranslationJobStatus;
    page?: number;
    page_size?: number;
    sort?: "newest" | "oldest";
  }): Promise<ListJobsResponse> => {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append("page", params.page.toString());
    if (params?.page_size) queryParams.append("page_size", params.page_size.toString());
    if (params?.status) queryParams.append("status", params.status);
    if (params?.sort) queryParams.append("sort", params.sort);

    const query = queryParams.toString();
    return client.get<ListJobsResponse>(`/api/v1/translation/jobs${query ? `?${query}` : ""}`);
  },

  getJob: (jobId: string): Promise<TranslationJob> =>
    client.get<TranslationJob>(`/api/v1/translation/jobs/${jobId}`),

  createJob: (data: CreateJobRequest): Promise<TranslationJob> =>
    client.post<TranslationJob>("/api/v1/translation/jobs", data),

  approveJob: (jobId: string): Promise<TranslationJob> =>
    client.post<TranslationJob>(`/api/v1/translation/jobs/${jobId}/approve`),

  rejectJob: (jobId: string, data?: RejectJobRequest): Promise<TranslationJob> =>
    client.post<TranslationJob>(`/api/v1/translation/jobs/${jobId}/reject`, data || {}),

  updateSegment: (
    jobId: string,
    segmentUuid: string,
    data: UpdateSegmentRequest
  ): Promise<TranslationSegment> =>
    client.put<TranslationSegment>(
      `/api/v1/translation/jobs/${jobId}/segments/${segmentUuid}`,
      data
    ),

  getFlaggedSegments: (jobId: string): Promise<FlaggedSegmentsResponse> =>
    client.get<FlaggedSegmentsResponse>(`/api/v1/translation/jobs/${jobId}/flagged`),
};
