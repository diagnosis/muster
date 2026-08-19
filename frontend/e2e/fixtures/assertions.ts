import {expect, type Page} from "@playwright/test";


export async function expectLoggedIn(page:Page, name:string){
    await openNav(page)
    await expect(page.getByRole('link', {name:'My outings'})).toBeVisible()
    await expect(page.getByRole('link', {name:'Create outing'})).toBeVisible()
    await expect(page.getByRole('link', {name:name})).toBeVisible()
}
export async function expectLoggedOut(page:Page){
    await openNav(page)
    await expect(page.getByRole('link', {name:'Log in'})).toBeVisible()
    await expect(page.getByRole('link', {name:'Sign up'})).toBeVisible()
}

const MOBILE_BREAKPOINT = 768   // wire-check against Header.module.css media query, then delete this note

export async function openNav(page: Page) {
    const vw = page.viewportSize()?.width ?? 1280
    if (vw >= MOBILE_BREAKPOINT) return
    const burger = page.getByRole('button', { name: 'Menu' })
    await expect(burger).toBeVisible()
    if (await burger.getAttribute('aria-expanded') === 'false') await burger.click()
    await expect(burger).toHaveAttribute('aria-expanded', 'true')
}

export async function expectNavClosed(page: Page) {
    const vw = page.viewportSize()?.width ?? 1280
    if (vw >= MOBILE_BREAKPOINT) return
    await expect(page.getByRole('button', { name: 'Menu' }))
        .toHaveAttribute('aria-expanded', 'false')
}
