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

type MintOverrides = {purpose?: string, ttlHours?: number, ttlMins?: number}
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

export async function setOutingTimes(outingID: string, startsAt: Date, endsAt: Date | null): Promise<void> {
    await getPool().query(
        `UPDATE outings SET starts_at = $2, ends_at = $3 WHERE id = $1`,
        [outingID, startsAt, endsAt],
    )
}