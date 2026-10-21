import { Pool } from 'pg'
import { createHash, randomBytes} from "node:crypto"

let pool : Pool | null = null

function getPool(): Pool {
    if (!pool) {
        const dsn = process.env.DATABASE_URL
        if (!dsn) throw new Error('DATABASE_URL not set - e2e db fixtures need it')
        pool = new Pool({connectionString:dsn})
    }
    return pool
}

// mirrors go-toolkit secure.HashRefreshToken: sha256, hex
function hashToken(raw: string): string {
    return createHash('sha256').update(raw).digest('hex')
}

type MintOverrides = {purpose?: string, ttlHours?: number, ttlMins?:number}
export async function mintVerificationToken(hikerID: string, overrides: Partial<MintOverrides> = {}): Promise<string>{
    const raw = randomBytes(32).toString('hex')
    await getPool().query(
        `INSERT INTO auth_tokens (id, hiker_id, token_hash, purpose, created_at, expires_at)
         VALUES (gen_random_uuid(), $1, $2, $3, now(), now() + make_interval(hours => $4))`,
        [hikerID, hashToken(raw), overrides.purpose ?? 'email_verification', overrides.ttlHours ?? 24],
    )
    return raw
}

export async function mintLoginCode(hikerID:string, overrides:Partial<MintOverrides> = {}){
    // mirrors authtoken.hashCode: sha256(hikerID + code)
    const raw = String(Math.floor(Math.random() * 1_000_000)).padStart(6, '0')
    await getPool().query(
        `INSERT INTO auth_tokens (id, hiker_id, token_hash, purpose, created_at, expires_at)
         VALUES (gen_random_uuid(), $1, $2, $3, now(), now() + make_interval(mins => $4))`,
        [hikerID, hashToken(hikerID+raw), overrides.purpose ?? 'login_code', overrides.ttlMins ?? 10],
    )
    return raw
}
export async function liveLoginCodeAttempts(hikerID: string): Promise<number> {
    const { rows } = await getPool().query(
        `SELECT attempts FROM auth_tokens
         WHERE hiker_id = $1 AND purpose = 'login_code' AND used_at IS NULL
         ORDER BY created_at DESC LIMIT 1`,
        [hikerID],
    )
    return rows[0].attempts
}

export async function getNotificationsFor(hikerID: string): Promise<{payload:Record<string, any>,kind: string, emailed_at: string|null}[]> {
    const res = await getPool().query(
        `SELECT payload, kind, emailed_at FROM notification_events WHERE hiker_id = $1 ORDER BY created_at`,
        [hikerID],
    )
    return res.rows
}

export async function getNotificationEmailedAt(hikerID: string): Promise<(string | null)[]> {
    const res = await getPool().query(
        `SELECT emailed_at FROM notification_events WHERE hiker_id = $1 ORDER BY created_at`,
        [hikerID],
    )
    return res.rows.map(r => r.emailed_at)
}

// setOutingTimes moves an outing's start and end directly in the database.
// The API refuses past times, so specs use this to test rules about outings
// that have already ended. ends_at must be later than starts_at, or null.
export async function setOutingTimes(outingID: string, startsAt: Date, endsAt: Date | null): Promise<void> {
    await getPool().query(
        `UPDATE outings SET starts_at = $2, ends_at = $3 WHERE id = $1`,
        [outingID, startsAt, endsAt],
    )
}