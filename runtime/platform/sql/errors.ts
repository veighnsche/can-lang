// Shared SQL failure construction. Every SQL factory builds its failures
// here so pools, transactions, and dialects report identical identities
// and fields: sanitized codes and names only, never messages, URLs, or
// credentials. Origins stay per-factory; this module only shapes payloads.
import { failure, type Completion } from "../../completion.ts";
import { record } from "../../data.ts";
import { createDomainRuntime } from "../../domain.ts";
import type { FailureOrigin } from "../../failure.ts";

export type SQLCoreContracts = Readonly<{
  connectionFailed: string;
  queryFailed: string;
  rowMissing: string;
  rowCount: string;
  schemaMismatch: string;
  constraintFailed: string;
  rowLimit: string;
  unsupportedValue: string;
}>;
export type SQLPoolContracts = SQLCoreContracts &
  Readonly<{
    credentialsMissing: string;
    closeFailed: string;
  }>;
export type SQLTxContracts = SQLCoreContracts &
  Readonly<{
    transactionFailed: string;
    commitUnknown: string;
  }>;

export type SQLFailures = {
  readonly fail: (
    identity: string,
    fields: readonly (readonly [string, unknown])[],
  ) => Completion<never>;
  readonly connectionFailed: (phase: string) => Completion<never>;
  readonly queryFailed: (operation: string, code: string) => Completion<never>;
  readonly mismatch: (path: string, reason: string) => Completion<never>;
  readonly badValue: (path: string, reason: string) => Completion<never>;
};

export function createSQLFailures(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: SQLCoreContracts,
  origin: FailureOrigin,
): SQLFailures {
  const fail = (
    identity: string,
    fields: readonly (readonly [string, unknown])[],
  ): Completion<never> => failure(domain.create(identity, record(identity, fields), origin));
  return {
    fail,
    connectionFailed: (phase: string) => fail(contracts.connectionFailed, [["phase", phase]]),
    queryFailed: (operation: string, code: string) =>
      fail(contracts.queryFailed, [
        ["operation", operation],
        ["code", code],
      ]),
    mismatch: (path: string, reason: string) =>
      fail(contracts.schemaMismatch, [
        ["path", path],
        ["reason", reason],
      ]),
    badValue: (path: string, reason: string) =>
      fail(contracts.unsupportedValue, [
        ["path", path],
        ["reason", reason],
      ]),
  };
}
