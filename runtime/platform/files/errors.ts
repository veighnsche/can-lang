import { types as nativeTypes } from "node:util";

// Verified native failure classification. Unknown throws and programming
// defects stay standard failures; only listed codes become domain errors.
export type FileErrorKind =
  | "not_found"
  | "denied"
  | "already_exists"
  | "not_empty"
  | "cross_device"
  | "unexpected_kind"
  | "not_directory"
  | "name_too_long"
  | "io_error";
const denied = new Set(["EACCES", "EPERM", "EROFS"]);
const ioCodes = new Set([
  "EIO",
  "EBUSY",
  "EFBIG",
  "EINTR",
  "EAGAIN",
  "ENXIO",
  "EBADF",
  "ELOOP",
  "EMFILE",
  "ENFILE",
  "ENOSPC",
  "EDQUOT",
  "ESTALE",
  "ETXTBSY",
  "EUCLEAN",
  "ENOMEM",
  "ENOTSUP",
]);
export function fileErrorCode(cause: unknown): string | undefined {
  if (
    cause === null ||
    typeof cause !== "object" ||
    nativeTypes.isProxy(cause) ||
    !nativeTypes.isNativeError(cause)
  )
    return undefined;
  const code = Object.getOwnPropertyDescriptor(cause, "code");
  if (code === undefined || !("value" in code) || typeof code.value !== "string") return undefined;
  return code.value;
}
export function classifyFileError(
  cause: unknown,
): { kind: FileErrorKind; code: string } | undefined {
  const code = fileErrorCode(cause);
  if (code === undefined) return undefined;
  if (code === "ENOENT") return { kind: "not_found", code };
  if (denied.has(code)) return { kind: "denied", code };
  if (code === "EEXIST") return { kind: "already_exists", code };
  if (code === "EISDIR" || code === "ERR_FS_EISDIR") return { kind: "unexpected_kind", code };
  if (code === "ENOTDIR") return { kind: "not_directory", code };
  if (code === "ENOTEMPTY") return { kind: "not_empty", code };
  if (code === "EXDEV") return { kind: "cross_device", code };
  if (code === "ENAMETOOLONG") return { kind: "name_too_long", code };
  if (ioCodes.has(code)) return { kind: "io_error", code };
  return undefined;
}
// Static path validation before any native call. Reasons feed
// files::invalid_path; the empty string and NUL bytes never reach the OS.
export function validatePath(path: string): string | undefined {
  if (path === "") return "empty";
  if (path.includes("\0")) return "nul_byte";
  return undefined;
}
export function entryKind(file: boolean, directory: boolean, symlink: boolean): string {
  if (symlink) return "symlink";
  if (file) return "file";
  if (directory) return "directory";
  return "other";
}
export type EntryShape = Readonly<{
  isFile(): boolean;
  isDirectory(): boolean;
  isSymbolicLink(): boolean;
}>;
export function kindOfEntry(entry: EntryShape): string {
  return entryKind(entry.isFile(), entry.isDirectory(), entry.isSymbolicLink());
}
