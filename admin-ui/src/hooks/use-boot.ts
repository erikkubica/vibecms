import { useQuery } from '@tanstack/react-query';
import type { BootManifest } from '../sdui/types';

async function fetchBoot(): Promise<BootManifest> {
  const res = await fetch('/admin/api/boot', { credentials: 'include' });
  if (!res.ok) throw new Error('Failed to fetch boot manifest');
  const json = await res.json();
  return json.data;
}

export function useBoot() {
  return useQuery({
    queryKey: ['boot'],
    queryFn: fetchBoot,
    staleTime: 60_000, // 1 minute — boot manifest changes rarely
  });
}
