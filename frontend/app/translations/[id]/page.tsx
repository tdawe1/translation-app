"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { useTranslationStore } from "@/store/translationStore";
import type { TranslationSegment } from "@/lib/api/types";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { ErrorBoundary } from "@/components/error-boundary";
import { SectionHeader } from "@/components/ui/base/SectionHeader";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";
import { toast } from "@/store/toast";
import { authApi } from "@/lib/api";
import { navigateToHome } from "@/lib/navigation";

export default function TranslationJobPage() {
  const params = useParams();
  const id = params?.id as string;
  const router = useRouter();
  const user = useAuthStore((state) => state.user);
  const logout = async () => {
    try {
      await authApi.logout();
    } catch (err) {
      // Just continue
    } finally {
      sessionStorage.removeItem("access_token");
      navigateToHome();
    }
  };

  const {
    currentJob,
    currentJobLoading,
    currentJobError,
    fetchJob,
    approveJob,
    rejectJob,
    updateSegment,
  } = useTranslationStore();

  const [filterFlagged, setFilterFlagged] = useState(false);
  const [editingSegmentId, setEditingSegmentId] = useState<string | null>(null);
  const [editedText, setEditedText] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (id) {
      fetchJob(id);
    }
  }, [id, fetchJob]);

  const handleApprove = async () => {
    if (!id) return;
    try {
      await approveJob(id);
      toast.success("Job approved successfully");
    } catch (error) {
      // Handled by store
    }
  };

  const handleReject = async () => {
    if (!id) return;
    try {
      await rejectJob(id);
      toast.success("Job rejected");
    } catch (error) {
      // Handled by store
    }
  };

  const startEditing = (segment: TranslationSegment) => {
    setEditingSegmentId(segment.id);
    setEditedText(segment.target || "");
  };

  const cancelEditing = () => {
    setEditingSegmentId(null);
    setEditedText("");
  };

  const saveSegment = async (segment: TranslationSegment) => {
    if (!id) return;
    setIsSaving(true);
    try {
      await updateSegment(id, segment.id, {
        target: editedText,
      });
      setEditingSegmentId(null);
      toast.success("Segment updated");
    } catch (error) {
      // Handled by store
    } finally {
      setIsSaving(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "approved":
      case "completed":
        return "text-green-600";
      case "rejected":
      case "failed":
      case "cancelled":
        return "text-red-600";
      case "pending_approval":
        return "text-orange-600";
      case "processing":
      case "translating":
        return "text-blue-600";
      default:
        return "text-neutral-600";
    }
  };

  const displayedSegments = currentJob?.segments || [];
  const filteredSegments = filterFlagged
    ? displayedSegments.filter((s: TranslationSegment) => s.is_flagged)
    : displayedSegments;

  if (currentJobLoading && !currentJob) {
    return (
      <div className="min-h-screen bg-neutral-50 flex items-center justify-center">
        <div className="animate-pulse flex flex-col items-center">
          <div className="h-4 w-32 bg-neutral-200 rounded mb-4"></div>
          <div className="h-4 w-24 bg-neutral-200 rounded"></div>
        </div>
      </div>
    );
  }

  if (currentJobError) {
    return (
      <div className="min-h-screen bg-neutral-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-xl font-light text-red-600 mb-2">Error Loading Job</h2>
          <p className="text-neutral-600">{currentJobError}</p>
          <Button
            className="mt-4"
            onClick={() => id && fetchJob(id)}
            variant="secondary"
          >
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <ProtectedRoute>
      <ErrorBoundary>
        <main className="min-h-screen bg-neutral-50">
          <header className="bg-white border-b border-neutral-200 sticky top-0 z-10">
            <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
              <div className="flex items-center gap-8">
                <Link
                  href="/"
                  className="text-xl font-light tracking-tighter text-neutral-900 hover:text-blue-600 transition-colors duration-150"
                >
                  GengoWatcher
                </Link>
                <nav className="hidden md:flex items-center gap-6">
                  <Link
                    href="/dashboard"
                    className="text-sm font-medium text-neutral-600 hover:text-neutral-900 transition-colors"
                  >
                    Dashboard
                  </Link>
                  <span className="text-neutral-300">/</span>
                  <span className="text-sm font-medium text-neutral-900">
                    Translation Review
                  </span>
                </nav>
              </div>
              <div className="flex items-center gap-6">
                <span className="hidden sm:block font-mono text-xs text-neutral-500 uppercase tracking-widest">
                  {user?.email}
                </span>
                <button
                  onClick={logout}
                  className="font-mono text-xs text-neutral-900 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150"
                >
                  Sign Out
                </button>
              </div>
            </div>
          </header>

          <div className="max-w-7xl mx-auto px-6 py-12">
            <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-6 mb-8">
              <div className="flex-1">
                <SectionHeader
                  title={currentJob?.source_file || "Translation Job"}
                  meta={`ID: ${id}`}
                  accentColor="blue"
                />
                <div className="mt-2 flex items-center gap-4 text-sm text-neutral-500 font-mono">
                  <span>{currentJob?.source_lang} → {currentJob?.target_lang}</span>
                  <span>•</span>
                  <span>{new Date(currentJob?.created_at || "").toLocaleDateString()}</span>
                </div>
              </div>
              
              {currentJob?.status === "pending_approval" && (
                <div className="flex items-center gap-3">
                  <Button
                    onClick={handleReject}
                    variant="secondary"
                    className="border-red-200 text-red-700 hover:bg-red-50 hover:border-red-300"
                  >
                    Reject Translation
                  </Button>
                  <Button
                    onClick={handleApprove}
                    variant="primary"
                    className="bg-green-600 hover:bg-green-700 border-green-600"
                  >
                    Approve Translation
                  </Button>
                </div>
              )}
            </div>

            <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-12">
              <BentoCard accentColor="blue" staggerIndex={0} className="p-6">
                <h3 className="font-mono text-xs uppercase tracking-widest text-blue-600 mb-2">
                  Status
                </h3>
                <p className={`text-2xl font-light capitalize ${getStatusColor(currentJob?.status || "")}`}>
                  {(currentJob?.status || "").replace("_", " ")}
                </p>
              </BentoCard>

              <BentoCard accentColor="violet" staggerIndex={1} className="p-6">
                <h3 className="font-mono text-xs uppercase tracking-widest text-violet-600 mb-2">
                  Quality Score
                </h3>
                <div className="flex items-end gap-2">
                  <p className="text-3xl font-light text-neutral-900">
                    {currentJob?.overall_score ?? "N/A"}
                  </p>
                  <span className="text-sm text-neutral-500 mb-1">/ 100</span>
                </div>
              </BentoCard>

              <BentoCard accentColor="green" staggerIndex={2} className="p-6">
                <h3 className="font-mono text-xs uppercase tracking-widest text-green-600 mb-2">
                  Progress
                </h3>
                <div className="relative pt-1">
                  <div className="flex mb-2 items-center justify-between">
                    <span className="text-3xl font-light text-neutral-900">
                      {currentJob?.progress ?? 0}%
                    </span>
                  </div>
                  <div className="overflow-hidden h-1 mb-4 text-xs flex rounded bg-green-100">
                    <div
                      style={{ width: `${currentJob?.progress ?? 0}%` }}
                      className="shadow-none flex flex-col text-center whitespace-nowrap text-white justify-center bg-green-500 transition-all duration-500"
                    ></div>
                  </div>
                </div>
              </BentoCard>

              <BentoCard accentColor="orange" staggerIndex={3} className="p-6">
                <h3 className="font-mono text-xs uppercase tracking-widest text-orange-600 mb-2">
                  Segments
                </h3>
                <div className="flex justify-between items-end">
                  <div>
                    <p className="text-3xl font-light text-neutral-900">
                      {currentJob?.segment_count ?? displayedSegments.length}
                    </p>
                    <p className="text-xs text-neutral-500 mt-1">Total</p>
                  </div>
                  <div className="text-right">
                    <p className="text-xl font-light text-orange-600">
                      {currentJob?.flagged_count ?? displayedSegments.filter((s: TranslationSegment) => s.is_flagged).length}
                    </p>
                    <p className="text-xs text-neutral-500 mt-1">Flagged</p>
                  </div>
                </div>
              </BentoCard>
            </div>

            <div className="space-y-6">
              <div className="flex items-center justify-between border-b border-neutral-200 pb-4">
                <h2 className="text-lg font-medium text-neutral-900">Translation Segments</h2>
                <div className="flex items-center gap-3">
                  <label className="flex items-center gap-2 text-sm text-neutral-600 cursor-pointer select-none">
                    <div className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={filterFlagged}
                        onChange={(e) => setFilterFlagged(e.target.checked)}
                      />
                      <div className="w-9 h-5 bg-neutral-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-100 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-neutral-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
                    </div>
                    <span>Show Flagged Only</span>
                  </label>
                </div>
              </div>

              <div className="space-y-4">
                {filteredSegments.length === 0 ? (
                  <div className="text-center py-12 bg-white border border-neutral-200 rounded-sm">
                    <p className="text-neutral-500">No segments found matching your filter.</p>
                  </div>
                ) : (
                  filteredSegments.map((segment: TranslationSegment) => (
                    <div
                      key={segment.id}
                      className={`bg-white border transition-colors duration-200 ${
                        segment.is_flagged ? "border-orange-200 bg-orange-50/10" : "border-neutral-200"
                      }`}
                    >
                      <div className="grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-neutral-100">
                        <div className="p-4 md:p-6">
                          <div className="flex items-center justify-between mb-3">
                            <span className="font-mono text-xs text-neutral-400 uppercase tracking-wider">
                              Source ({currentJob?.source_lang})
                            </span>
                            {segment.is_flagged && (
                              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-800">
                                Flagged
                              </span>
                            )}
                          </div>
                          <p className="text-neutral-800 leading-relaxed whitespace-pre-wrap">
                            {segment.source}
                          </p>
                        </div>

                        <div className="p-4 md:p-6 bg-neutral-50/50">
                          <div className="flex items-center justify-between mb-3">
                            <span className="font-mono text-xs text-neutral-400 uppercase tracking-wider">
                              Target ({currentJob?.target_lang})
                            </span>
                            {editingSegmentId === segment.id ? (
                              <div className="flex items-center gap-2">
                                <button
                                  onClick={cancelEditing}
                                  className="text-xs text-neutral-500 hover:text-neutral-900 px-2 py-1"
                                >
                                  Cancel
                                </button>
                                <button
                                  onClick={() => saveSegment(segment)}
                                  disabled={isSaving}
                                  className="text-xs bg-blue-600 text-white px-3 py-1 hover:bg-blue-700 transition-colors disabled:opacity-50"
                                >
                                  {isSaving ? "Saving..." : "Save"}
                                </button>
                              </div>
                            ) : (
                              <button
                                onClick={() => startEditing(segment)}
                                className="text-xs text-blue-600 hover:text-blue-800 font-medium px-2 py-1"
                              >
                                Edit
                              </button>
                            )}
                          </div>
                          
                          {editingSegmentId === segment.id ? (
                            <textarea
                              value={editedText}
                              onChange={(e) => setEditedText(e.target.value)}
                              className="w-full min-h-[100px] p-3 text-neutral-800 bg-white border border-blue-300 outline-none focus:ring-2 focus:ring-blue-100 rounded-sm resize-y"
                              placeholder="Enter translation..."
                              autoFocus
                            />
                          ) : (
                            <p 
                              className="text-neutral-800 leading-relaxed whitespace-pre-wrap cursor-text hover:bg-blue-50/50 -m-1 p-1 rounded transition-colors"
                              onClick={() => startEditing(segment)}
                            >
                              {segment.target || <span className="text-neutral-400 italic">No translation yet</span>}
                            </p>
                          )}
                          
                          {segment.flag_reason && (
                            <div className="mt-4 p-3 bg-orange-50 border border-orange-100 text-sm text-orange-800 rounded-sm">
                              <span className="font-medium">Flag reason:</span> {segment.flag_reason}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </main>
      </ErrorBoundary>
    </ProtectedRoute>
  );
}
