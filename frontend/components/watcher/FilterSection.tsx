import { cn } from "@/lib/utils";
import { DESIGN } from "@/lib/design/tokens";

interface FilterSectionProps {
  label: string;
  children: React.ReactNode;
}

export function FilterSection({ label, children }: FilterSectionProps) {
  return (
    <div>
      <label className={cn("block mb-2", DESIGN.typography.label)}>
        {label}
      </label>
      {children}
    </div>
  );
}
