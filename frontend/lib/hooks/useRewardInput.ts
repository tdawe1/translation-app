import { useState, useCallback } from "react";

interface UseRewardInputProps {
  minReward: number | null;
  maxReward: number | null;
  onRewardChange: (min: number | null, max: number | null) => void;
}

export function useRewardInput({ minReward, maxReward, onRewardChange }: UseRewardInputProps) {
  const [minInput, setMinInput] = useState(minReward?.toString() ?? "");
  const [maxInput, setMaxInput] = useState(maxReward?.toString() ?? "");

  const applyRewardRange = useCallback(() => {
    const min = minInput ? parseFloat(minInput) : null;
    const max = maxInput ? parseFloat(maxInput) : null;
    onRewardChange(min, max);
  }, [minInput, maxInput, onRewardChange]);

  const setMinReward = useCallback((min: number) => {
    setMinInput(min.toString());
    onRewardChange(min, maxReward);
  }, [maxReward, onRewardChange]);

  const setMaxReward = useCallback((max: number) => {
    setMaxInput(max.toString());
    onRewardChange(minReward, max);
  }, [minReward, onRewardChange]);

  return {
    minInput,
    maxInput,
    setMinInput,
    setMaxInput,
    applyRewardRange,
    setMinReward,
    setMaxReward,
  };
}
