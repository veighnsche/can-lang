# B1-05 operation/error table

As-built contract for the `stream` catalogue package and the file stream
producers. Native causes are retained privately on domain failures;
public payloads carry only the fields below.

## Operations

| Operation | Native basis | Public errors |
|---|---|---|
| stream::read_many(reader, max_items) | one ReadableStream pull per batch step | read_failed, cancelled, files::limit_exceeded |
| stream::write_some(writer, chunk) | FileSink.write accepted count | write_failed |
| stream::close_reader(reader) | reader cancel + lock release, owner terminal | close_failed |
| stream::close_writer(writer) | sink flush + end, owner terminal | close_failed |
| stream::cancel_reader(reader, reason) | reason record + owner terminal | close_failed |
| stream::cancel_writer(writer, reason) | reason record + owner terminal | close_failed |
| files::read_stream(path, max_chunk) | stat at open, lazy Bun.file stream, split items | files::not_found, denied, invalid_path, unexpected_kind, limit_exceeded, io_error |
| files::read_lines_stream(path, max_line) | stat at open, fatal streaming decode, \n framing | files::not_found, denied, invalid_path, unexpected_kind, limit_exceeded, io_error |
| files::write_stream(path) | create/truncate at open, lazy FileSink | files::not_found, denied, invalid_path, unexpected_kind, io_error |

## Lifecycle rules

States open -> closing -> closed with exactly one terminal publication,
kept by owner.ts. Empty batch is the normal end and is sticky until
close. Read failures (native error, malformed UTF-8, line over cap) are
terminal for the reader: the failing call reports the domain failure and
later calls observe standard resource-state. Close still succeeds after
a read failure so failure arms can clean up. Cancel is terminal and
records its reason: a read interrupted by cancel reports cancelled and
delivers no partial items. Close twice, use-after-close and
foreign-owner handles throw standard resource-state failures, exactly
like every other owner violation; they are never domain failures.

## Error mapping

| Public error | Fields | Cause |
|---|---|---|
| stream::read_failed (1316) | reason | native read rejection (reason is the native code or io_error), malformed UTF-8 (reason utf8) |
| stream::write_failed (1317) | reason | native write rejection, malformed accepted count (reason bad_accepted) |
| stream::cancelled (1318) | reason | read interrupted by cancel; caller-supplied reason |
| stream::close_failed (1319) | reason | native terminal failure during close/cancel (reason close), bounded-shutdown timeout (reason close_timeout) |
| files::limit_exceeded | limit | max_items/max_chunk/max_line below 1, line over max_line |

## Framing policy

Bytes readers split source chunks at max_chunk; callers choose batch
size per read. Line readers split on \n, strip one trailing \r, deliver
a final unterminated segment as a line, and decode fatal UTF-8 with BOM
ignored, matching whole-file reads. Line caps count raw retained bytes
per line plus an exact encoded check per emitted line.
