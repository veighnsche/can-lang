import { record } from "../data.ts";
import { copyBytes, ownBytes } from "../bytes.ts";
import { invoke, success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { providerHTTP, rawEnvironment } from "../assert/provider.ts";
import { createTransport, type HTTPTypes } from "../transport/http.ts";
import { createNormalizer } from "../transport/normalize.ts";
import type { Connection } from "../transport/request.ts";
import type { Schema } from "../codec/json.ts";
import { CodecIssue } from "../codec/budget.ts";

import {
  encodeQuestions,
  decodeAnswers,
  QuestionIssue,
  AnswerIssue,
  type QuestionDescriptor,
  type Answer,
} from "./questions.ts";
export {
  encodeQuestions,
  decodeAnswers,
  validateQuestions,
  QuestionIssue,
  AnswerIssue,
  type QuestionDescriptor,
  type NoulDescriptor,
  type Answer,
} from "./questions.ts";

export type AITypes = HTTPTypes &
  Readonly<{ invalidData: string; invalidQuestion: string; invalidAnswer: string; failed: string }>;
// BudgetBinding mirrors the server-native GuardedConnection from
// ./budget.ts structurally so this shipped module gains no new import
// edge. The binding carries one guarded provider connection: every
// call on a bound factory reserves through the R14 ledger before
// sending, and calls without scope context fail closed.
export type BudgetContextInput = Readonly<{ tenant: string; pool: string; correlation: string }>;
export type BudgetBinding = Readonly<{
  dispatch: <T>(
    attempt: Readonly<{
      context: BudgetContextInput | undefined;
      where: FailureOrigin;
      operation: string;
      maxBodyBytes: number;
      send: () => Promise<Readonly<{ completion: Completion<T>; body: Uint8Array | undefined }>>;
    }>,
  ) => Promise<Completion<T>>;
}>;
export function createTypeSafe(
  domain: ReturnType<typeof createDomainRuntime>,
  types: AITypes,
  readEnvironment: (name: string) => string | undefined,
  budget?: BudgetBinding,
) {
  const transport = createTransport(domain, types, readEnvironment);
  const normalize = createNormalizer(domain, {
    failed: types.failed,
    leaves: [
      types.invalid,
      types.credential,
      types.transport,
      types.timeout,
      types.limit,
      types.status,
      types.invalidData,
    ],
  });
  function issue(
    cause: unknown,
    origin: FailureOrigin,
    operation: string,
    requestLimit?: number,
  ): Completion<never> {
    let identity: string, fields: readonly (readonly [string, unknown])[];
    let boundary: "native" | "emitted" = "emitted";
    if (cause instanceof QuestionIssue) {
      identity = types.invalidQuestion;
      fields = [["reason", cause.reason]];
    } else if (cause instanceof AnswerIssue) {
      identity = types.invalidAnswer;
      fields = [
        ["question", cause.question],
        ["reason", cause.reason],
      ];
    } else if (cause instanceof CodecIssue) {
      boundary = "native";
      if (cause.reason === "byte_limit" && requestLimit !== undefined) {
        identity = types.limit;
        fields = [["limit", BigInt(requestLimit)]];
      } else {
        identity = types.invalidData;
        fields = [
          ["path", cause.path],
          ["reason", cause.reason],
        ];
      }
    } else throw cause;
    return failure(
      domain.create(identity, record(identity, fields), origin, undefined, { boundary, operation }),
    );
  }
  return Object.freeze({
    async ask(
      connection: Connection,
      model: string,
      stateSchema: Schema,
      state: unknown,
      questions: readonly QuestionDescriptor[],
      origin: FailureOrigin,
      operation: string,
      context?: AssertionContext,
      budgetContext?: BudgetContextInput,
    ): Promise<Completion<readonly Answer[]>> {
      return normalize.map(
        await invoke(async () => {
          let body: Uint8Array;
          try {
            body = copyBytes(
              encodeQuestions(model, stateSchema, state, questions, connection.maxBodyBytes),
              origin,
            );
          } catch (cause) {
            return issue(cause, origin, operation, connection.maxBodyBytes);
          }
          const exchange = providerHTTP(context, origin, operation, connection.maxBodyBytes);
          if (exchange === undefined) denyLiveBoundary(context, origin);
          const respond = (bytes: Uint8Array): Completion<readonly Answer[]> => {
            try {
              return success(decodeAnswers(ownBytes(bytes), questions, connection.maxBodyBytes));
            } catch (cause) {
              return issue(cause, origin, operation);
            }
          };
          const readEnv = rawEnvironment(context, operation) ?? readEnvironment;
          if (budget === undefined)
            return transport.request(
              connection,
              {
                path: "",
                method: "POST",
                query: [],
                headers: [
                  { name: "content_type", value: "application/json" },
                  { name: "accept", value: "application/json" },
                ],
                body,
                exchange,
              },
              respond,
              origin,
              operation,
              readEnv,
            );
          // Guarded connections reserve before sending and settle
          // authoritative usage after; the raw body copy lets the
          // guard decode usage even when the answers themselves fail
          // to decode. Budget failures are not normalizer leaves, so
          // they pass through untouched below.
          let responseBody: Uint8Array | undefined;
          return budget.dispatch({
            context: budgetContext,
            where: origin,
            operation,
            maxBodyBytes: connection.maxBodyBytes,
            send: async () => ({
              completion: await transport.request(
                connection,
                {
                  path: "",
                  method: "POST",
                  query: [],
                  headers: [
                    { name: "content_type", value: "application/json" },
                    { name: "accept", value: "application/json" },
                  ],
                  body,
                  exchange,
                },
                (bytes) => {
                  responseBody = bytes.slice();
                  return respond(bytes);
                },
                origin,
                operation,
                readEnv,
              ),
              body: responseBody,
            }),
          });
        }, origin),
        operation,
      );
    },
  });
}
