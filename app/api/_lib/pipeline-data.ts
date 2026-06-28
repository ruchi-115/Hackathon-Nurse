export class BackendRequestError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "BackendRequestError";
    this.status = status;
  }
}

const backendBase = process.env.GO_API_BASE_URL ?? "http://localhost:8080";

export async function fetchBackend<T>(
  route: string,
  init?: RequestInit
): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 1200);
  try {
    const res = await fetch(`${backendBase}${route}`, {
      ...init,
      cache: "no-store",
      signal: controller.signal
    });

    if (!res.ok) {
      throw new BackendRequestError(
        res.status,
        `Go backend returned ${res.status}`
      );
    }

    return (await res.json()) as T;
  } catch (error) {
    if (error instanceof BackendRequestError) {
      throw error;
    }
    throw new BackendRequestError(
      502,
      "Unable to reach Go backend at http://localhost:8080"
    );
  } finally {
    clearTimeout(timeout);
  }
}

export function backendErrorResponse(error: unknown) {
  if (error instanceof BackendRequestError) {
    return Response.json({ error: error.message }, { status: error.status });
  }
  return Response.json(
    { error: "Unable to reach Go backend" },
    { status: 502 }
  );
}
