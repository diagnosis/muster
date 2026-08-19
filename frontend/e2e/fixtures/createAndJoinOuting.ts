import type {CreateOutingInput, JoinRequestInput} from "./types.ts";
import {createActor} from "./actor.ts";
import {createOuting, joinRequest} from "./outingApiHelper.ts";

export async function createAndJoinOuting(outingOverride:Partial<CreateOutingInput> = {}, joiningOverrides:Partial<JoinRequestInput> = {}){
    const actorHost = await createActor()
    const actorHiker = await createActor()
    const outing = await createOuting(actorHost, outingOverride)
    const jr = await joinRequest(actorHiker, outing.id, joiningOverrides)
    return {actorHost, actorHiker, outing, jr}
}