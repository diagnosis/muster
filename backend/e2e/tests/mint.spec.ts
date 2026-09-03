import { test } from '@playwright/test'
import {mintVerificationToken} from "../db";

test.skip('generate raw', async ()=> {
    //just replace id
    const id = "4caf522c-62a7-47c1-8e1d-c28efc3b8820"
    console.log(await mintVerificationToken(id))
})