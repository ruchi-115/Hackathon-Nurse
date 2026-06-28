import { promises as fs } from "fs";
import path from "path";
import type { SyncSnapshot } from "./types";

const DATA_DIR = path.join(process.cwd(), "data");
const SNAPSHOT_PATH = path.join(DATA_DIR, "sync-snapshot.json");

export async function saveSnapshot(snapshot: SyncSnapshot) {
  await fs.mkdir(DATA_DIR, { recursive: true });
  await fs.writeFile(SNAPSHOT_PATH, JSON.stringify(snapshot, null, 2), "utf8");
}

export async function loadSnapshot(): Promise<SyncSnapshot | null> {
  try {
    const raw = await fs.readFile(SNAPSHOT_PATH, "utf8");
    return JSON.parse(raw) as SyncSnapshot;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      return null;
    }
    throw error;
  }
}

export function snapshotPath() {
  return SNAPSHOT_PATH;
}
