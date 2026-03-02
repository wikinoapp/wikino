import { closePool } from "./helpers/database";

async function globalTeardown() {
  await closePool();
}

export default globalTeardown;
