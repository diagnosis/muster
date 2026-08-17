import {expect, type Page} from "@playwright/test";


export async function expectLoggedIn(page:Page, name:string){
    await expect(page.getByRole('link', {name:'My outings'})).toBeVisible()
    await expect(page.getByRole('link', {name:'Create outing'})).toBeVisible()
    await expect(page.getByRole('link', {name:name})).toBeVisible()
}
export async function expectLoggedOut(page:Page){
    await expect(page.getByRole('link', {name:'Log in'})).toBeVisible()
    await expect(page.getByRole('link', {name:'Sign up'})).toBeVisible()
}