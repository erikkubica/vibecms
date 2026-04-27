import React from "react";

const {
  ListToolbar,
  ListSearch,
  Input,
} = (window as any).__VIBECMS_SHARED__.ui;

export interface ToolbarFilters {
  search: string;
  status: string;
  dateFrom: string;
  dateTo: string;
}

interface SubmissionsToolbarProps {
  filters: ToolbarFilters;
  searchValue: string;
  onSearchChange: (v: string) => void;
  onChange: (next: Partial<ToolbarFilters>) => void;
}

export default function SubmissionsToolbar({
  filters,
  searchValue,
  onSearchChange,
  onChange,
}: SubmissionsToolbarProps) {
  return (
    <ListToolbar>
        <ListSearch
          value={searchValue}
          onChange={onSearchChange}
          placeholder="Search submissions…"
        />

        <div className="flex items-center gap-1.5">
          <label className="text-[12px] text-slate-500 whitespace-nowrap">From</label>
          <Input
            type="date"
            className="h-[30px] w-36 text-[13px] bg-white border-slate-300 rounded"
            value={filters.dateFrom}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              onChange({ dateFrom: e.target.value })
            }
          />
        </div>

        <div className="flex items-center gap-1.5">
          <label className="text-[12px] text-slate-500 whitespace-nowrap">To</label>
          <Input
            type="date"
            className="h-[30px] w-36 text-[13px] bg-white border-slate-300 rounded"
            value={filters.dateTo}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              onChange({ dateTo: e.target.value })
            }
          />
        </div>
    </ListToolbar>
  );
}
