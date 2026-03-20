import type {
  BikChallenge,
  BikGame,
  BikGameResult,
  CreateChallengeActivity,
  AcceptChallengeActivity,
  UndoChallengeActivity,
  CreateGameActivity,
  CreateGameResultActivity,
} from "./types/ap.js";

const CONTEXT = "https://www.w3.org/ns/activitystreams" as const;

export function buildCreateChallenge(
  actor: string,
  id: string,
  challenge: BikChallenge,
  content: string,
): CreateChallengeActivity {
  return {
    "@context": CONTEXT,
    type: "Create",
    actor,
    object: { type: "Note", id, attributedTo: actor, content, attachment: challenge },
  };
}

export function buildAcceptChallenge(
  actor: string,
  challengeUri: string,
): AcceptChallengeActivity {
  return { "@context": CONTEXT, type: "Accept", actor, object: challengeUri };
}

export function buildUndoChallenge(
  actor: string,
  challengeUri: string,
): UndoChallengeActivity {
  return { "@context": CONTEXT, type: "Undo", actor, object: challengeUri };
}

export function buildCreateGame(
  actor: string,
  id: string,
  challengeUri: string,
  game: BikGame,
  content: string,
): CreateGameActivity {
  return {
    "@context": CONTEXT,
    type: "Create",
    actor,
    object: { type: "Note", id, content, inReplyTo: challengeUri, attachment: game },
  };
}

export function buildCreateGameResult(
  actor: string,
  id: string,
  gameUri: string,
  result: BikGameResult,
  content: string,
): CreateGameResultActivity {
  return {
    "@context": CONTEXT,
    type: "Create",
    actor,
    object: { type: "Note", id, content, inReplyTo: gameUri, attachment: result },
  };
}

/** POST an AP activity to a remote inbox */
export async function postActivity(inboxUrl: string, activity: object): Promise<Response> {
  return fetch(inboxUrl, {
    method: "POST",
    headers: { "Content-Type": "application/activity+json" },
    body: JSON.stringify(activity),
  });
}
