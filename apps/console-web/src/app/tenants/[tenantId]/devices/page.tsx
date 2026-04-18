"use client";

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function DevicesRedirect({ params }: { params: { tenantId: string } }) {
  const router = useRouter();
  useEffect(() => {
    router.replace(`/tenants/${params.tenantId}/health`);
  }, [router, params.tenantId]);
  return null;
}
