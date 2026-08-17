

const W = Number(process.env.TEST_WORKER_INDEX ?? 0).toString(36)
let seq = 0
export function tag(): string {
    return W + (seq++ % 1296).toString(36).padStart(2, '0') + Math.random().toString(36).slice(2, 5)
}