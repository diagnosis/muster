import {expect, type Page} from "@playwright/test";
import type {CreateOutingInput} from "./types.ts";

const W = Number(process.env.TEST_WORKER_INDEX ?? 0).toString(36)
let seq = 0
export function tag(): string {
    return W + (seq++ % 1296).toString(36).padStart(2, '0') + Math.random().toString(36).slice(2, 5)
}

export async function fillCreateForm(page: Page, outing:CreateOutingInput){
    await expect(page.getByRole('heading', { name: 'Create Outing' })).toBeVisible()

    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('textbox', {name: 'Title'}).fill(outing.title)
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('textbox', { name: 'Destination' }).fill(outing.destination)
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('textbox', { name: 'Meet Label' }).fill(outing.meet_label)
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('textbox', { name: 'Starts At' }).fill(datetimeLocal(48))
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('spinbutton', { name: 'Max Size' }).fill(outing.max_size.toString())
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('spinbutton', { name: 'Host Seats' }).fill(outing.host_seats.toString())
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
    await page.getByRole('spinbutton', { name: 'Cost per Seat' }).fill((outing.cost_per_seat_cents/100).toString())
    await page.getByRole('group', { name: 'Difficulty' }).getByText(outing.difficulty).click()
    await page.getByRole('group', { name: 'Pace' }).getByText(outing.pace).click()
    await expect(page.getByRole('button', {name:'Create Outing'})).toBeEnabled()
    await page.getByRole('textbox', { name: 'Notes' }).fill(outing.notes?outing.notes:"")
}

function datetimeLocal(hoursFromNow: number): string {
    const d = new Date(Date.now() + hoursFromNow * 3_600_000)
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset())  // shift so toISOString prints local wall-clock
    return d.toISOString().slice(0, 16)                    // "2026-08-16T21:30"
}