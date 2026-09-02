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

type MintOverrides = {purpose?: string, ttlHours?: number}
export async function mintVerificationToken(hikerID: string, overrides: Partial<MintOverrides> = {}): Promise<string>{
    const raw = randomBytes(32).toString('hex')
    await getPool().query(
        `INSERT INTO auth_tokens (id, hiker_id, token_hash, purpose, created_at, expires_at)
         VALUES (gen_random_uuid(), $1, $2, $3, now(), now() + make_interval(hours => $4))`,
        [hikerID, hashToken(raw), overrides.purpose ?? 'email_verification', overrides.ttlHours ?? 24],
    )
    return raw
}