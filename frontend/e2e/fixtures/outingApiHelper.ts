// e2e/fixtures/outingApiHelper.ts


import type {CreateOutingInput, JoinRequest, JoinRequestInput, Outing} from "./types.ts";
import {type Actor} from "./actor.ts";
import {tag} from "./mint.ts";
import {unwrap} from "./api.ts";
const HOURS = [48, 72, 96, 120]
const outings: Omit<CreateOutingInput, 'starts_at'>[] = [
    { title: 'Poo Poo Point', destination: 'Issaquah', meet_label: 'South Bellevue P&R', max_size: 10, host_seats: 3, cost_per_seat_cents: 300, difficulty: 'moderate', pace: 'relaxed', notes: 'Paraglider launch views at the top' },
    { title: 'Rattlesnake Ledge', destination: 'North Bend', meet_label: 'Eastgate P&R', max_size: 8, host_seats: 4, cost_per_seat_cents: 2500, difficulty: 'easy', pace: 'relaxed', notes: 'Busy trail — early start' },
    { title: 'Mount Si', destination: 'North Bend', meet_label: 'Issaquah Highlands P&R', max_size: 6, host_seats: 3, cost_per_seat_cents: 400, difficulty: 'hard', pace: 'moderate', notes: 'Bring plenty of water, 3100 ft gain' },
    { title: 'Little Si', destination: 'North Bend', meet_label: 'Eastgate P&R', max_size: 8, host_seats: 4, cost_per_seat_cents: 0, difficulty: 'easy', pace: 'relaxed', notes: 'Good first hike of the season' },
    { title: 'Mailbox Peak', destination: 'North Bend', meet_label: 'South Bellevue P&R', max_size: 4, host_seats: 4, cost_per_seat_cents: 500, difficulty: 'hard', pace: 'fast', notes: 'New trail up, old trail down for the brave' },
    { title: 'Twin Falls', destination: 'Olallie State Park', meet_label: 'Eastgate P&R', max_size: 10, host_seats: 2, cost_per_seat_cents: 0, difficulty: 'easy', pace: 'relaxed', notes: 'Waterfall viewpoint, kid-friendly' },
    { title: 'Snow Lake', destination: 'Snoqualmie Pass', meet_label: 'Issaquah TC', max_size: 8, host_seats: 3, cost_per_seat_cents: 400, difficulty: 'moderate', pace: 'moderate', notes: 'Alpine lake, lingering snow in early season' },
    { title: 'Franklin Falls', destination: 'Snoqualmie Pass', meet_label: 'North Bend P&R', max_size: 12, host_seats: 4, cost_per_seat_cents: 200, difficulty: 'easy', pace: 'relaxed', notes: 'Short walk, big waterfall' },
    { title: 'Granite Mountain', destination: 'Snoqualmie Pass', meet_label: 'Eastgate P&R', max_size: 6, host_seats: 3, cost_per_seat_cents: 500, difficulty: 'hard', pace: 'fast', notes: 'Fire lookout at the summit, avalanche risk in spring' },
    { title: 'Melakwa Lake', destination: 'Denny Creek', meet_label: 'Issaquah Highlands P&R', max_size: 6, host_seats: 4, cost_per_seat_cents: 400, difficulty: 'hard', pace: 'moderate', notes: 'Cross the famous waterslide slabs' },
    { title: 'Lake Serene', destination: 'Index', meet_label: 'Lynnwood TC', max_size: 8, host_seats: 4, cost_per_seat_cents: 600, difficulty: 'hard', pace: 'moderate', notes: 'Bridal Veil Falls spur is worth it' },
    { title: 'Wallace Falls', destination: 'Gold Bar', meet_label: 'Lynnwood TC', max_size: 10, host_seats: 3, cost_per_seat_cents: 500, difficulty: 'moderate', pace: 'relaxed', notes: 'Three tiers of falls, well-graded trail' },
    { title: 'Heybrook Lookout', destination: 'Index', meet_label: 'Monroe P&R', max_size: 8, host_seats: 4, cost_per_seat_cents: 350, difficulty: 'easy', pace: 'relaxed', notes: 'Short climb to a staffed fire lookout' },
    { title: 'Lake 22', destination: 'Mountain Loop Highway', meet_label: 'Everett Station', max_size: 6, host_seats: 3, cost_per_seat_cents: 500, difficulty: 'moderate', pace: 'moderate', notes: 'Old-growth cedar and a cirque lake' },
    { title: 'Heather Lake', destination: 'Mountain Loop Highway', meet_label: 'Everett Station', max_size: 8, host_seats: 4, cost_per_seat_cents: 500, difficulty: 'moderate', pace: 'relaxed', notes: 'Boardwalk loop around the lake' },
    { title: 'Mount Pilchuck', destination: 'Granite Falls', meet_label: 'Lynnwood TC', max_size: 6, host_seats: 3, cost_per_seat_cents: 600, difficulty: 'hard', pace: 'moderate', notes: 'Lookout scramble at the top — gloves help' },
    { title: 'Big Four Ice Caves', destination: 'Mountain Loop Highway', meet_label: 'Everett Station', max_size: 12, host_seats: 4, cost_per_seat_cents: 400, difficulty: 'easy', pace: 'relaxed', notes: 'View caves from a distance — never enter' },
    { title: 'Skyline Trail', destination: 'Mount Rainier — Paradise', meet_label: 'Tacoma Dome Station', max_size: 8, host_seats: 4, cost_per_seat_cents: 900, difficulty: 'hard', pace: 'moderate', notes: 'Wildflower season is unreal, park pass needed' },
    { title: 'Tolmie Peak', destination: 'Mount Rainier — Mowich', meet_label: 'Puyallup P&R', max_size: 6, host_seats: 3, cost_per_seat_cents: 800, difficulty: 'moderate', pace: 'moderate', notes: 'Eunice Lake + lookout views of Rainier' },
    { title: 'Naches Peak Loop', destination: 'Chinook Pass', meet_label: 'Enumclaw P&R', max_size: 8, host_seats: 4, cost_per_seat_cents: 700, difficulty: 'easy', pace: 'relaxed', notes: 'Loop clockwise for Rainier views' },
    { title: 'Colchuck Lake', destination: 'Leavenworth', meet_label: 'Monroe P&R', max_size: 6, host_seats: 3, cost_per_seat_cents: 1000, difficulty: 'hard', pace: 'fast', notes: 'Enchantments gateway — permits for overnight only' },
    { title: 'Icicle Ridge', destination: 'Leavenworth', meet_label: 'Monroe P&R', max_size: 8, host_seats: 4, cost_per_seat_cents: 950, difficulty: 'moderate', pace: 'moderate', notes: 'Switchbacks with Leavenworth views' },
    { title: 'Maple Pass Loop', destination: 'North Cascades', meet_label: 'Everett Station', max_size: 6, host_seats: 3, cost_per_seat_cents: 1200, difficulty: 'hard', pace: 'moderate', notes: 'Best larches in October' },
    { title: 'Blue Lake', destination: 'North Cascades', meet_label: 'Everett Station', max_size: 8, host_seats: 4, cost_per_seat_cents: 1100, difficulty: 'easy', pace: 'relaxed', notes: 'Short alpine gem off Highway 20' },
    { title: 'Cascade Pass', destination: 'North Cascades', meet_label: 'Burlington P&R', max_size: 6, host_seats: 3, cost_per_seat_cents: 1200, difficulty: 'moderate', pace: 'moderate', notes: 'Gravel road to trailhead — high clearance helps' },
    { title: 'Oyster Dome', destination: 'Chuckanut Mountains', meet_label: 'Burlington P&R', max_size: 8, host_seats: 4, cost_per_seat_cents: 600, difficulty: 'moderate', pace: 'moderate', notes: 'Samish Bay overlook at the top' },
    { title: 'Ebey\'s Landing', destination: 'Whidbey Island', meet_label: 'Mukilteo Ferry', max_size: 10, host_seats: 4, cost_per_seat_cents: 700, difficulty: 'easy', pace: 'relaxed', notes: 'Bluff loop over the Sound — ferry ride included in the fun' },
    { title: 'Dungeness Spit', destination: 'Sequim', meet_label: 'Edmonds Ferry', max_size: 8, host_seats: 4, cost_per_seat_cents: 900, difficulty: 'easy', pace: 'relaxed', notes: 'Longest sand spit in the US — check tides' },
    { title: 'Mount Storm King', destination: 'Lake Crescent', meet_label: 'Edmonds Ferry', max_size: 4, host_seats: 4, cost_per_seat_cents: 1100, difficulty: 'hard', pace: 'fast', notes: 'Rope-assisted final scramble — confident hikers only' },
    { title: 'Hurricane Hill', destination: 'Olympic National Park', meet_label: 'Edmonds Ferry', max_size: 8, host_seats: 3, cost_per_seat_cents: 1000, difficulty: 'moderate', pace: 'relaxed', notes: 'Paved-ish path, panoramic Olympics' },
]

function hoursFromNowISO(hours: number): string{
    const date = new Date()
    date.setTime(date.getTime() + hours * 60 * 60 * 1000)
    return date.toISOString();
}



export function uniqueOuting(overrides: Partial<CreateOutingInput> = {}): CreateOutingInput{
    // pick random outing
    const randomOutingIndex = Math.floor(Math.random() * outings.length);
    const pickedOuting = outings[randomOutingIndex]
    // pick random hour
    const randomHourIndex = Math.floor(Math.random() * HOURS.length)
    const pickedHourOffset = HOURS[randomHourIndex]

    return {
        ...pickedOuting,
        title: `${pickedOuting.title} ${tag()}`,
        starts_at: hoursFromNowISO(pickedHourOffset),
        ...overrides
    }
}

export async function createOuting(actor: Actor, overrides: Partial<CreateOutingInput>= {}) : Promise<Outing>{
    const res = await actor.api.post('/api/outings', {data: uniqueOuting(overrides)})
    return unwrap<Outing>(res, 'create outing')
}

export async function joinRequest(actor:Actor, outingId:string, overrides:Partial<JoinRequestInput>= {}) : Promise<JoinRequest>{
    const body: JoinRequestInput = { role: 'rider', seats_offered: 0, guests: 0, ...overrides }
    const res = await actor.api.post(`/api/outings/${outingId}/requests`, { data: body })
    return unwrap<JoinRequest>(res, 'join request')
}

export async function cancelOuting(actor:Actor, outingId: string){
    const res = await actor.api.post(`/api/outings/${outingId}/cancel`)
    return unwrap(res, 'cancel outing')
}

export async function acceptRequest(actor:Actor, requestId: string){
    const res = await actor.api.post(`/api/requests/${requestId}/accept`)
    return unwrap<JoinRequest>(res, 'accept request')
}

export async function declineRequest(actor:Actor, requestId: string){
    const res = await  actor.api.post(`/api/requests/${requestId}/decline`)
    return unwrap<JoinRequest>(res, 'decline request')
}