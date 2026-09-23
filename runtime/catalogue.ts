// Code generated from compiler/internal/catalogue/catalogue.json; DO NOT EDIT.
import { dataArray, dataKeys, dataProperty } from "./data.ts";
function freeze<T>(value: T): Readonly<T> {
  if (value !== null && typeof value === 'object') {
    for (const child of Object.values(value)) freeze(child);
    Object.freeze(value);
  }
  return value;
}
export const catalogueSHA256 = "e4b64d70c19a181494f2d60456bc70d026a903e0593337caccd7b85045e127a3";
export const catalogue = freeze({
  "schemaVersion": 1,
  "revision": 1,
  "targetId": "bun-1.4.2-darwin-arm64-v1",
  "packages": [
    {
      "name": "ai",
      "identity": "can.std.ai@1"
    },
    {
      "name": "asset",
      "identity": "can.std.asset@1"
    },
    {
      "name": "bytes",
      "identity": "can.std.bytes@1"
    },
    {
      "name": "checks",
      "identity": "can.std.checks@1"
    },
    {
      "name": "cli",
      "identity": "can.std.cli@1"
    },
    {
      "name": "clock",
      "identity": "can.std.clock@1"
    },
    {
      "name": "codec",
      "identity": "can.std.codec@1"
    },
    {
      "name": "collections",
      "identity": "can.std.collections@1"
    },
    {
      "name": "cookie",
      "identity": "can.std.cookie@1"
    },
    {
      "name": "crypto",
      "identity": "can.std.crypto@1"
    },
    {
      "name": "csrf",
      "identity": "can.std.csrf@1"
    },
    {
      "name": "env",
      "identity": "can.std.env@1"
    },
    {
      "name": "files",
      "identity": "can.std.files@1"
    },
    {
      "name": "html",
      "identity": "can.std.html@1"
    },
    {
      "name": "htmx",
      "identity": "can.std.htmx@1"
    },
    {
      "name": "http",
      "identity": "can.std.http@1"
    },
    {
      "name": "io",
      "identity": "can.std.io@1"
    },
    {
      "name": "json",
      "identity": "can.std.json@1"
    },
    {
      "name": "llm",
      "identity": "can.std.llm@1"
    },
    {
      "name": "log",
      "identity": "can.std.log@1"
    },
    {
      "name": "number",
      "identity": "can.std.number@1"
    },
    {
      "name": "option",
      "identity": "can.std.option@1"
    },
    {
      "name": "password",
      "identity": "can.std.password@1"
    },
    {
      "name": "path",
      "identity": "can.std.path@1"
    },
    {
      "name": "process",
      "identity": "can.std.process@1"
    },
    {
      "name": "random",
      "identity": "can.std.random@1"
    },
    {
      "name": "sql",
      "identity": "can.std.sql@1"
    },
    {
      "name": "stream",
      "identity": "can.std.stream@1"
    },
    {
      "name": "text",
      "identity": "can.std.text@1"
    },
    {
      "name": "time",
      "identity": "can.std.time@1"
    },
    {
      "name": "url",
      "identity": "can.std.url@1"
    },
    {
      "name": "ws",
      "identity": "can.std.ws@1"
    }
  ],
  "prelude": [
    "append",
    "choice_option",
    "all_failed",
    "standard_failure"
  ],
  "standardFailures": [
    "arithmetic",
    "bounds",
    "resource_state",
    "assertion",
    "native_exception",
    "cleanup"
  ],
  "types": [
    {
      "name": "choice_option",
      "identity": "can.prelude@1::choice_option",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "key",
          "type": "str"
        },
        {
          "name": "description",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "standard_failure",
      "identity": "can.prelude@1::standard_failure",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [
        {
          "name": "occurrence_id",
          "type": "int"
        },
        {
          "name": "kind",
          "type": "str"
        },
        {
          "name": "message",
          "type": "str"
        }
      ],
      "constructible": false
    },
    {
      "name": "option::none",
      "identity": "can.std.option@1::none",
      "kind": "record",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "option::some",
      "identity": "can.std.option@1::some",
      "kind": "record",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [
        {
          "name": "value",
          "type": "T"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "option::value",
      "identity": "can.std.option@1::value",
      "kind": "variant",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [],
      "leaves": [
        "option::none",
        "option::some<T>"
      ],
      "projections": [],
      "constructible": false
    },
    {
      "name": "number::division",
      "identity": "can.std.number@1::division",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "quotient",
          "type": "int"
        },
        {
          "name": "remainder",
          "type": "int"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "number::rounded",
      "identity": "can.std.number@1::rounded",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "value",
          "type": "int"
        },
        {
          "name": "remainder_numerator",
          "type": "int"
        },
        {
          "name": "denominator",
          "type": "int"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "collections::entry",
      "identity": "can.std.collections@1::entry",
      "kind": "record",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "fields": [
        {
          "name": "key",
          "type": "K"
        },
        {
          "name": "value",
          "type": "V"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "collections::map",
      "identity": "can.std.collections@1::map",
      "kind": "opaque",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "collections::set",
      "identity": "can.std.collections@1::set",
      "kind": "opaque",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::header",
      "identity": "can.std.http@1::header",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "http::sse_event",
      "identity": "can.std.http@1::sse_event",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "data",
          "type": "str"
        },
        {
          "name": "event",
          "type": "str"
        },
        {
          "name": "id",
          "type": "str"
        },
        {
          "name": "retry",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "http::multipart_form",
      "identity": "can.std.http@1::multipart_form",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "fields",
          "type": "http::multipart_field[]"
        },
        {
          "name": "files",
          "type": "http::multipart_file[]"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "http::multipart_field",
      "identity": "can.std.http@1::multipart_field",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "http::multipart_file",
      "identity": "can.std.http@1::multipart_file",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "filename",
          "type": "str"
        },
        {
          "name": "content_type",
          "type": "str"
        },
        {
          "name": "content",
          "type": "bytes::buffer"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "http::response",
      "identity": "can.std.http@1::response",
      "kind": "record",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [
        {
          "name": "status",
          "type": "int"
        },
        {
          "name": "headers",
          "type": "http::header[]"
        },
        {
          "name": "body",
          "type": "T"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "http::failure_detail",
      "identity": "can.std.http@1::failure_detail",
      "kind": "variant",
      "parameters": [],
      "fields": [],
      "leaves": [
        "http::invalid_request",
        "http::credentials_missing",
        "http::transport_failed",
        "http::timeout",
        "http::body_limit",
        "http::status_error",
        "codec::invalid_data"
      ],
      "projections": [],
      "constructible": false
    },
    {
      "name": "bytes::buffer",
      "identity": "can.std.bytes@1::buffer",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "html::node",
      "identity": "can.std.html@1::node",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "html::safe",
      "identity": "can.std.html@1::safe",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "html::url",
      "identity": "can.std.html@1::url",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "html::attribute",
      "identity": "can.std.html@1::attribute",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "html::tag",
      "identity": "can.std.html@1::tag",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "htmx::target",
      "identity": "can.std.htmx@1::target",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "htmx::swap",
      "identity": "can.std.htmx@1::swap",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::request",
      "identity": "can.std.http@1::request",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::server_response",
      "identity": "can.std.http@1::server_response",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::status",
      "identity": "can.std.http@1::status",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::body_status",
      "identity": "can.std.http@1::body_status",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::server_headers",
      "identity": "can.std.http@1::server_headers",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::route",
      "identity": "can.std.http@1::route",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::router",
      "identity": "can.std.http@1::router",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::server",
      "identity": "can.std.http@1::server",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::server_config",
      "identity": "can.std.http@1::server_config",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "http::tls_config",
      "identity": "can.std.http@1::tls_config",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "sql::pool",
      "identity": "can.std.sql@1::pool",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "sql::transaction",
      "identity": "can.std.sql@1::transaction",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "sql::sqlite_file_options",
      "identity": "can.std.sql@1::sqlite_file_options",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "mode",
          "type": "str"
        },
        {
          "name": "busy_timeout_ms",
          "type": "int"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "sql::commit",
      "identity": "can.std.sql@1::commit",
      "kind": "record",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [
        {
          "name": "value",
          "type": "T"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "sql::rollback",
      "identity": "can.std.sql@1::rollback",
      "kind": "record",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [
        {
          "name": "value",
          "type": "T"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "sql::decision",
      "identity": "can.std.sql@1::decision",
      "kind": "variant",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [],
      "leaves": [
        "sql::commit<T>",
        "sql::rollback<T>"
      ],
      "projections": [],
      "constructible": false
    },
    {
      "name": "files::file_info",
      "identity": "can.std.files@1::file_info",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "kind",
          "type": "str"
        },
        {
          "name": "size",
          "type": "int"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "files::entry",
      "identity": "can.std.files@1::entry",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "kind",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "process::options",
      "identity": "can.std.process@1::options",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "cwd",
          "type": "str"
        },
        {
          "name": "inherit_env",
          "type": "bool"
        },
        {
          "name": "env",
          "type": "str[]"
        },
        {
          "name": "stdin",
          "type": "bytes::buffer"
        },
        {
          "name": "stdout_limit",
          "type": "int"
        },
        {
          "name": "stderr_limit",
          "type": "int"
        },
        {
          "name": "deadline_ms",
          "type": "int"
        },
        {
          "name": "grace_ms",
          "type": "int"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "process::result",
      "identity": "can.std.process@1::result",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "stdout",
          "type": "bytes::buffer"
        },
        {
          "name": "stderr",
          "type": "bytes::buffer"
        },
        {
          "name": "code",
          "type": "int"
        },
        {
          "name": "signal",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "stream::reader",
      "identity": "can.std.stream@1::reader",
      "kind": "opaque",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "stream::writer",
      "identity": "can.std.stream@1::writer",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "crypto::key",
      "identity": "can.std.crypto@1::key",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "crypto::keypair",
      "identity": "can.std.crypto@1::keypair",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "private_key",
          "type": "crypto::key"
        },
        {
          "name": "public_key",
          "type": "crypto::key"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "crypto::sealed",
      "identity": "can.std.crypto@1::sealed",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "nonce",
          "type": "bytes::buffer"
        },
        {
          "name": "ciphertext",
          "type": "bytes::buffer"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "url::parts",
      "identity": "can.std.url@1::parts",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "scheme",
          "type": "str"
        },
        {
          "name": "host",
          "type": "str"
        },
        {
          "name": "port",
          "type": "int"
        },
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "query",
          "type": "str"
        },
        {
          "name": "fragment",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "url::query_pair",
      "identity": "can.std.url@1::query_pair",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "text::regex",
      "identity": "can.std.text@1::regex",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "text::regex_match",
      "identity": "can.std.text@1::regex_match",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "text",
          "type": "str"
        },
        {
          "name": "start",
          "type": "int"
        },
        {
          "name": "end",
          "type": "int"
        },
        {
          "name": "groups",
          "type": "str[]"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "time::instant",
      "identity": "can.std.time@1::instant",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "time::civil",
      "identity": "can.std.time@1::civil",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "year",
          "type": "int"
        },
        {
          "name": "month",
          "type": "int"
        },
        {
          "name": "day",
          "type": "int"
        },
        {
          "name": "hour",
          "type": "int"
        },
        {
          "name": "minute",
          "type": "int"
        },
        {
          "name": "second",
          "type": "int"
        },
        {
          "name": "millisecond",
          "type": "int"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "ws::session",
      "identity": "can.std.ws@1::session",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "ws::connection",
      "identity": "can.std.ws@1::connection",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "session",
          "type": "ws::session"
        },
        {
          "name": "events",
          "type": "stream::reader<ws::event>"
        },
        {
          "name": "protocol",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "ws::event",
      "identity": "can.std.ws@1::event",
      "kind": "variant",
      "parameters": [],
      "fields": [],
      "leaves": [
        "ws::text",
        "ws::binary",
        "ws::drain",
        "ws::closed"
      ],
      "projections": [],
      "constructible": false
    },
    {
      "name": "ws::text",
      "identity": "can.std.ws@1::text",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "text",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "ws::binary",
      "identity": "can.std.ws@1::binary",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "data",
          "type": "bytes::buffer"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "ws::drain",
      "identity": "can.std.ws@1::drain",
      "kind": "record",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "ws::closed",
      "identity": "can.std.ws@1::closed",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "code",
          "type": "int"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "cookie::attributes",
      "identity": "can.std.cookie@1::attributes",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "domain",
          "type": "option::value<str>"
        },
        {
          "name": "secure",
          "type": "bool"
        },
        {
          "name": "http_only",
          "type": "bool"
        },
        {
          "name": "same_site",
          "type": "cookie::same_site"
        },
        {
          "name": "max_age",
          "type": "option::value<int>"
        },
        {
          "name": "expires_ms",
          "type": "option::value<int>"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "cookie::same_site",
      "identity": "can.std.cookie@1::same_site",
      "kind": "variant",
      "parameters": [],
      "fields": [],
      "leaves": [
        "cookie::strict",
        "cookie::lax",
        "cookie::none"
      ],
      "projections": [],
      "constructible": false
    },
    {
      "name": "cookie::strict",
      "identity": "can.std.cookie@1::strict",
      "kind": "record",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "cookie::lax",
      "identity": "can.std.cookie@1::lax",
      "kind": "record",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "cookie::none",
      "identity": "can.std.cookie@1::none",
      "kind": "record",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "cookie::cookie",
      "identity": "can.std.cookie@1::cookie",
      "kind": "opaque",
      "parameters": [],
      "fields": [],
      "leaves": [],
      "projections": [],
      "constructible": false
    },
    {
      "name": "cookie::collection",
      "identity": "can.std.cookie@1::collection",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "pairs",
          "type": "cookie::pair[]"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    },
    {
      "name": "cookie::pair",
      "identity": "can.std.cookie@1::pair",
      "kind": "record",
      "parameters": [],
      "fields": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        }
      ],
      "leaves": [],
      "projections": [],
      "constructible": true
    }
  ],
  "errors": [
    {
      "id": 100,
      "name": "all_failed",
      "identity": "can.prelude@1::all_failed",
      "parameters": [
        {
          "name": "F",
          "constraint": "failure_variant"
        }
      ],
      "fields": [
        {
          "name": "failures",
          "type": "F[]"
        }
      ]
    },
    {
      "id": 1000,
      "name": "number::inexact",
      "identity": "can.std.number@1::inexact",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1001,
      "name": "text::invalid_number",
      "identity": "can.std.text@1::invalid_number",
      "parameters": [],
      "fields": [
        {
          "name": "input",
          "type": "str"
        }
      ]
    },
    {
      "id": 1002,
      "name": "text::invalid_bool",
      "identity": "can.std.text@1::invalid_bool",
      "parameters": [],
      "fields": [
        {
          "name": "input",
          "type": "str"
        }
      ]
    },
    {
      "id": 1003,
      "name": "number::invalid_bool",
      "identity": "can.std.number@1::invalid_bool",
      "parameters": [],
      "fields": [
        {
          "name": "value",
          "type": "int"
        }
      ]
    },
    {
      "id": 1004,
      "name": "text::empty_separator",
      "identity": "can.std.text@1::empty_separator",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1005,
      "name": "text::empty_pattern",
      "identity": "can.std.text@1::empty_pattern",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1006,
      "name": "text::invalid_unicode",
      "identity": "can.std.text@1::invalid_unicode",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1007,
      "name": "collections::key_absent",
      "identity": "can.std.collections@1::key_absent",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1008,
      "name": "collections::key_exists",
      "identity": "can.std.collections@1::key_exists",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1009,
      "name": "number::zero_divisor",
      "identity": "can.std.number@1::zero_divisor",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1010,
      "name": "checks::failed",
      "identity": "can.std.checks@1::failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1100,
      "name": "http::invalid_request",
      "identity": "can.std.http@1::invalid_request",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1101,
      "name": "http::credentials_missing",
      "identity": "can.std.http@1::credentials_missing",
      "parameters": [],
      "fields": [
        {
          "name": "variable",
          "type": "str"
        }
      ]
    },
    {
      "id": 1102,
      "name": "http::transport_failed",
      "identity": "can.std.http@1::transport_failed",
      "parameters": [],
      "fields": [
        {
          "name": "phase",
          "type": "str"
        }
      ]
    },
    {
      "id": 1103,
      "name": "http::timeout",
      "identity": "can.std.http@1::timeout",
      "parameters": [],
      "fields": [
        {
          "name": "timeout_ms",
          "type": "int"
        }
      ]
    },
    {
      "id": 1104,
      "name": "http::body_limit",
      "identity": "can.std.http@1::body_limit",
      "parameters": [],
      "fields": [
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1105,
      "name": "http::status_error",
      "identity": "can.std.http@1::status_error",
      "parameters": [],
      "fields": [
        {
          "name": "status",
          "type": "int"
        },
        {
          "name": "headers",
          "type": "http::header[]"
        }
      ]
    },
    {
      "id": 1106,
      "name": "http::request_failed",
      "identity": "can.std.http@1::request_failed",
      "parameters": [],
      "fields": [
        {
          "name": "detail",
          "type": "http::failure_detail"
        }
      ]
    },
    {
      "id": 1110,
      "name": "codec::invalid_data",
      "identity": "can.std.codec@1::invalid_data",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1120,
      "name": "ai::invalid_question",
      "identity": "can.std.ai@1::invalid_question",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1121,
      "name": "ai::invalid_answer",
      "identity": "can.std.ai@1::invalid_answer",
      "parameters": [],
      "fields": [
        {
          "name": "question",
          "type": "str"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1130,
      "name": "llm::refused",
      "identity": "can.std.llm@1::refused",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1131,
      "name": "llm::truncated",
      "identity": "can.std.llm@1::truncated",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1132,
      "name": "llm::invalid_response",
      "identity": "can.std.llm@1::invalid_response",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1210,
      "name": "io::read_failed",
      "identity": "can.std.io@1::read_failed",
      "parameters": [],
      "fields": [
        {
          "name": "operation",
          "type": "str"
        }
      ]
    },
    {
      "id": 1211,
      "name": "io::write_failed",
      "identity": "can.std.io@1::write_failed",
      "parameters": [],
      "fields": [
        {
          "name": "operation",
          "type": "str"
        }
      ]
    },
    {
      "id": 1212,
      "name": "io::limit_exceeded",
      "identity": "can.std.io@1::limit_exceeded",
      "parameters": [],
      "fields": [
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1220,
      "name": "html::invalid_structure",
      "identity": "can.std.html@1::invalid_structure",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1221,
      "name": "html::invalid_url",
      "identity": "can.std.html@1::invalid_url",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1222,
      "name": "htmx::invalid_target",
      "identity": "can.std.htmx@1::invalid_target",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1223,
      "name": "htmx::invalid_interval",
      "identity": "can.std.htmx@1::invalid_interval",
      "parameters": [],
      "fields": [
        {
          "name": "milliseconds",
          "type": "int"
        }
      ]
    },
    {
      "id": 1230,
      "name": "http::invalid_route",
      "identity": "can.std.http@1::invalid_route",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1231,
      "name": "http::duplicate_route",
      "identity": "can.std.http@1::duplicate_route",
      "parameters": [],
      "fields": [
        {
          "name": "method",
          "type": "str"
        },
        {
          "name": "path",
          "type": "str"
        }
      ]
    },
    {
      "id": 1232,
      "name": "http::ambiguous_route",
      "identity": "can.std.http@1::ambiguous_route",
      "parameters": [],
      "fields": [
        {
          "name": "first",
          "type": "str"
        },
        {
          "name": "second",
          "type": "str"
        }
      ]
    },
    {
      "id": 1233,
      "name": "http::invalid_server_config",
      "identity": "can.std.http@1::invalid_server_config",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1234,
      "name": "http::bind_failed",
      "identity": "can.std.http@1::bind_failed",
      "parameters": [],
      "fields": [
        {
          "name": "address",
          "type": "str"
        }
      ]
    },
    {
      "id": 1235,
      "name": "http::shutdown_failed",
      "identity": "can.std.http@1::shutdown_failed",
      "parameters": [],
      "fields": [
        {
          "name": "phase",
          "type": "str"
        }
      ]
    },
    {
      "id": 1240,
      "name": "sql::connection_failed",
      "identity": "can.std.sql@1::connection_failed",
      "parameters": [],
      "fields": [
        {
          "name": "phase",
          "type": "str"
        }
      ]
    },
    {
      "id": 1241,
      "name": "sql::query_failed",
      "identity": "can.std.sql@1::query_failed",
      "parameters": [],
      "fields": [
        {
          "name": "operation",
          "type": "str"
        },
        {
          "name": "code",
          "type": "str"
        }
      ]
    },
    {
      "id": 1242,
      "name": "sql::row_missing",
      "identity": "can.std.sql@1::row_missing",
      "parameters": [],
      "fields": [
        {
          "name": "query",
          "type": "str"
        }
      ]
    },
    {
      "id": 1243,
      "name": "sql::row_count",
      "identity": "can.std.sql@1::row_count",
      "parameters": [],
      "fields": [
        {
          "name": "query",
          "type": "str"
        },
        {
          "name": "actual",
          "type": "int"
        }
      ]
    },
    {
      "id": 1244,
      "name": "sql::schema_mismatch",
      "identity": "can.std.sql@1::schema_mismatch",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1245,
      "name": "sql::constraint_failed",
      "identity": "can.std.sql@1::constraint_failed",
      "parameters": [],
      "fields": [
        {
          "name": "constraint",
          "type": "str"
        }
      ]
    },
    {
      "id": 1246,
      "name": "sql::transaction_failed",
      "identity": "can.std.sql@1::transaction_failed",
      "parameters": [],
      "fields": [
        {
          "name": "phase",
          "type": "str"
        }
      ]
    },
    {
      "id": 1247,
      "name": "sql::commit_unknown",
      "identity": "can.std.sql@1::commit_unknown",
      "parameters": [],
      "fields": [
        {
          "name": "transaction_id",
          "type": "str"
        }
      ]
    },
    {
      "id": 1248,
      "name": "sql::close_failed",
      "identity": "can.std.sql@1::close_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1249,
      "name": "sql::row_limit",
      "identity": "can.std.sql@1::row_limit",
      "parameters": [],
      "fields": [
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1250,
      "name": "sql::unsupported_value",
      "identity": "can.std.sql@1::unsupported_value",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1260,
      "name": "clock::invalid_duration",
      "identity": "can.std.clock@1::invalid_duration",
      "parameters": [],
      "fields": [
        {
          "name": "milliseconds",
          "type": "int"
        }
      ]
    },
    {
      "id": 1261,
      "name": "random::invalid_length",
      "identity": "can.std.random@1::invalid_length",
      "parameters": [],
      "fields": [
        {
          "name": "length",
          "type": "int"
        }
      ]
    },
    {
      "id": 1262,
      "name": "env::invalid_name",
      "identity": "can.std.env@1::invalid_name",
      "parameters": [],
      "fields": [
        {
          "name": "name",
          "type": "str"
        }
      ]
    },
    {
      "id": 1263,
      "name": "log::write_failed",
      "identity": "can.std.log@1::write_failed",
      "parameters": [],
      "fields": [
        {
          "name": "level",
          "type": "str"
        }
      ]
    },
    {
      "id": 1300,
      "name": "files::not_found",
      "identity": "can.std.files@1::not_found",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        }
      ]
    },
    {
      "id": 1301,
      "name": "files::denied",
      "identity": "can.std.files@1::denied",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "operation",
          "type": "str"
        }
      ]
    },
    {
      "id": 1302,
      "name": "files::already_exists",
      "identity": "can.std.files@1::already_exists",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        }
      ]
    },
    {
      "id": 1303,
      "name": "files::invalid_path",
      "identity": "can.std.files@1::invalid_path",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1304,
      "name": "files::io_error",
      "identity": "can.std.files@1::io_error",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "operation",
          "type": "str"
        }
      ]
    },
    {
      "id": 1305,
      "name": "files::limit_exceeded",
      "identity": "can.std.files@1::limit_exceeded",
      "parameters": [],
      "fields": [
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1306,
      "name": "files::not_empty",
      "identity": "can.std.files@1::not_empty",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        }
      ]
    },
    {
      "id": 1307,
      "name": "files::cross_device",
      "identity": "can.std.files@1::cross_device",
      "parameters": [],
      "fields": [
        {
          "name": "source",
          "type": "str"
        },
        {
          "name": "destination",
          "type": "str"
        }
      ]
    },
    {
      "id": 1308,
      "name": "files::unexpected_kind",
      "identity": "can.std.files@1::unexpected_kind",
      "parameters": [],
      "fields": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "operation",
          "type": "str"
        }
      ]
    },
    {
      "id": 1310,
      "name": "process::spawn_failed",
      "identity": "can.std.process@1::spawn_failed",
      "parameters": [],
      "fields": [
        {
          "name": "executable",
          "type": "str"
        }
      ]
    },
    {
      "id": 1311,
      "name": "process::timeout",
      "identity": "can.std.process@1::timeout",
      "parameters": [],
      "fields": [
        {
          "name": "deadline_ms",
          "type": "int"
        }
      ]
    },
    {
      "id": 1312,
      "name": "process::output_limit",
      "identity": "can.std.process@1::output_limit",
      "parameters": [],
      "fields": [
        {
          "name": "stream",
          "type": "str"
        },
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1313,
      "name": "process::nonzero",
      "identity": "can.std.process@1::nonzero",
      "parameters": [],
      "fields": [
        {
          "name": "code",
          "type": "int"
        },
        {
          "name": "signal",
          "type": "str"
        }
      ]
    },
    {
      "id": 1314,
      "name": "process::invalid_config",
      "identity": "can.std.process@1::invalid_config",
      "parameters": [],
      "fields": [
        {
          "name": "field",
          "type": "str"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1315,
      "name": "process::io_error",
      "identity": "can.std.process@1::io_error",
      "parameters": [],
      "fields": [
        {
          "name": "operation",
          "type": "str"
        }
      ]
    },
    {
      "id": 1316,
      "name": "stream::read_failed",
      "identity": "can.std.stream@1::read_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1317,
      "name": "stream::write_failed",
      "identity": "can.std.stream@1::write_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1318,
      "name": "stream::cancelled",
      "identity": "can.std.stream@1::cancelled",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1319,
      "name": "stream::close_failed",
      "identity": "can.std.stream@1::close_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1320,
      "name": "password::cost_rejected",
      "identity": "can.std.password@1::cost_rejected",
      "parameters": [],
      "fields": [
        {
          "name": "profile",
          "type": "int"
        }
      ]
    },
    {
      "id": 1321,
      "name": "password::invalid_hash",
      "identity": "can.std.password@1::invalid_hash",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1322,
      "name": "crypto::invalid_key",
      "identity": "can.std.crypto@1::invalid_key",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1323,
      "name": "crypto::invalid_nonce",
      "identity": "can.std.crypto@1::invalid_nonce",
      "parameters": [],
      "fields": [
        {
          "name": "length",
          "type": "int"
        }
      ]
    },
    {
      "id": 1324,
      "name": "crypto::key_misuse",
      "identity": "can.std.crypto@1::key_misuse",
      "parameters": [],
      "fields": [
        {
          "name": "operation",
          "type": "str"
        },
        {
          "name": "algorithm",
          "type": "str"
        }
      ]
    },
    {
      "id": 1325,
      "name": "crypto::decrypt_failed",
      "identity": "can.std.crypto@1::decrypt_failed",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1326,
      "name": "url::invalid_url",
      "identity": "can.std.url@1::invalid_url",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1327,
      "name": "text::invalid_regex",
      "identity": "can.std.text@1::invalid_regex",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1328,
      "name": "text::invalid_limit",
      "identity": "can.std.text@1::invalid_limit",
      "parameters": [],
      "fields": [
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1329,
      "name": "time::out_of_range",
      "identity": "can.std.time@1::out_of_range",
      "parameters": [],
      "fields": [
        {
          "name": "millis",
          "type": "int"
        }
      ]
    },
    {
      "id": 1330,
      "name": "time::invalid_zone",
      "identity": "can.std.time@1::invalid_zone",
      "parameters": [],
      "fields": [
        {
          "name": "zone",
          "type": "str"
        }
      ]
    },
    {
      "id": 1331,
      "name": "time::nonexistent_time",
      "identity": "can.std.time@1::nonexistent_time",
      "parameters": [],
      "fields": []
    },
    {
      "id": 1332,
      "name": "time::invalid_option",
      "identity": "can.std.time@1::invalid_option",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1333,
      "name": "ws::connect_failed",
      "identity": "can.std.ws@1::connect_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1334,
      "name": "ws::upgrade_failed",
      "identity": "can.std.ws@1::upgrade_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1335,
      "name": "ws::unsupported_protocol",
      "identity": "can.std.ws@1::unsupported_protocol",
      "parameters": [],
      "fields": [
        {
          "name": "protocol",
          "type": "str"
        }
      ]
    },
    {
      "id": 1336,
      "name": "ws::send_failed",
      "identity": "can.std.ws@1::send_failed",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1337,
      "name": "ws::invalid_close",
      "identity": "can.std.ws@1::invalid_close",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1338,
      "name": "ws::limit_exceeded",
      "identity": "can.std.ws@1::limit_exceeded",
      "parameters": [],
      "fields": [
        {
          "name": "limit",
          "type": "int"
        }
      ]
    },
    {
      "id": 1339,
      "name": "ws::invalid_url",
      "identity": "can.std.ws@1::invalid_url",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1340,
      "name": "ws::invalid_protocol",
      "identity": "can.std.ws@1::invalid_protocol",
      "parameters": [],
      "fields": [
        {
          "name": "protocol",
          "type": "str"
        }
      ]
    },
    {
      "id": 1341,
      "name": "cookie::invalid_cookie",
      "identity": "can.std.cookie@1::invalid_cookie",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    },
    {
      "id": 1342,
      "name": "csrf::invalid_config",
      "identity": "can.std.csrf@1::invalid_config",
      "parameters": [],
      "fields": [
        {
          "name": "reason",
          "type": "str"
        }
      ]
    }
  ],
  "operations": [
    {
      "name": "text::from_int",
      "identity": "can.std.text@1::from_int",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String"
        ],
        "adapter": "Native formatting without coercion at source boundaries.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "text::from_float",
      "identity": "can.std.text@1::from_float",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String"
        ],
        "adapter": "Native formatting without coercion at source boundaries.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "text::from_bool",
      "identity": "can.std.text@1::from_bool",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String"
        ],
        "adapter": "Native formatting without coercion at source boundaries.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::int_to_float",
      "identity": "can.std.number@1::int_to_float",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [
        "number::inexact"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number",
          "BigInt"
        ],
        "adapter": "Validate exactness, finiteness or the 0/1 boolean range before conversion.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::float_to_int",
      "identity": "can.std.number@1::float_to_int",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "number::inexact"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number",
          "BigInt"
        ],
        "adapter": "Validate exactness, finiteness or the 0/1 boolean range before conversion.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::bool_to_int",
      "identity": "can.std.number@1::bool_to_int",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number",
          "BigInt"
        ],
        "adapter": "Validate exactness, finiteness or the 0/1 boolean range before conversion.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::int_to_bool",
      "identity": "can.std.number@1::int_to_bool",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [
        "number::invalid_bool"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number",
          "BigInt"
        ],
        "adapter": "Validate exactness, finiteness or the 0/1 boolean range before conversion.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::floor",
      "identity": "can.std.number@1::floor",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Math.floor"
        ],
        "adapter": "Preserve native IEEE outcomes and signed zero.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::ceil",
      "identity": "can.std.number@1::ceil",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Math.ceil"
        ],
        "adapter": "Preserve native IEEE outcomes and signed zero.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::trunc",
      "identity": "can.std.number@1::trunc",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Math.trunc"
        ],
        "adapter": "Preserve native IEEE outcomes and signed zero.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::round",
      "identity": "can.std.number@1::round",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Math.round"
        ],
        "adapter": "Preserve native IEEE outcomes and signed zero.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::is_finite",
      "identity": "can.std.number@1::is_finite",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number.isFinite"
        ],
        "adapter": "Preserve native IEEE outcomes and signed zero.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "number::is_nan",
      "identity": "can.std.number@1::is_nan",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "float"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number.isNaN"
        ],
        "adapter": "Preserve native IEEE outcomes and signed zero.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "text::to_int",
      "identity": "can.std.text@1::to_int",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "text::invalid_number"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "BigInt"
        ],
        "adapter": "Full-match the specified grammar; reject nonfinite parsed floats.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "text::to_float",
      "identity": "can.std.text@1::to_float",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [
        "text::invalid_number"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Full-match the specified grammar; reject nonfinite parsed floats.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "text::to_bool",
      "identity": "can.std.text@1::to_bool",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [
        "text::invalid_bool"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "==="
        ],
        "adapter": "Full-match the specified grammar; reject nonfinite parsed floats.",
        "task": "I22"
      },
      "assertion": "real",
      "refs": [
        "C6"
      ]
    },
    {
      "name": "array.map",
      "identity": "can.intrinsic.array@1::map",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        },
        {
          "name": "U",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "U[]",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "U",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.fromAsync",
          "Array.prototype.keys",
          "Array.prototype.map"
        ],
        "adapter": "Iterate indices sequentially, box callback output, then unwrap synchronously.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.filter",
      "identity": "can.intrinsic.array@1::filter",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "bool",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.fromAsync",
          "Array.prototype.filter"
        ],
        "adapter": "Collect ordered boolean decisions before native filtering.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.for_each",
      "identity": "can.intrinsic.array@1::for_each",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "void",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.prototype.reduce"
        ],
        "adapter": "Await each callback through a native reduce chain, stopping on failure.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.fold",
      "identity": "can.intrinsic.array@1::fold",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        },
        {
          "name": "U",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "initial",
          "type": "U"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "U",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "U",
            "T"
          ],
          "result": "U",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.prototype.reduce"
        ],
        "adapter": "Box and await each accumulator with the explicit initial value.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.find",
      "identity": "can.intrinsic.array@1::find",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "option::value<T>",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "bool",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.prototype.values"
        ],
        "adapter": "Bounded ascending await adapter stops on the first true predicate.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.some",
      "identity": "can.intrinsic.array@1::some",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "bool",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.prototype.values"
        ],
        "adapter": "Bounded ascending await adapter stops on true; empty returns false.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.every",
      "identity": "can.intrinsic.array@1::every",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "bool",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.prototype.values"
        ],
        "adapter": "Bounded ascending await adapter stops on false; empty returns true.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.sort_by",
      "identity": "can.intrinsic.array@1::sort_by",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        },
        {
          "name": "K",
          "constraint": "sort_key"
        }
      ],
      "inputs": [
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "T"
          ],
          "result": "K",
          "deriveErrors": true,
          "emits": []
        }
      ],
      "emits": [],
      "callbackErrors": [
        "callback"
      ],
      "lowering": {
        "native": [
          "Array.fromAsync",
          "Array.prototype.toSorted"
        ],
        "adapter": "Evaluate each key once, reject nonfinite keys, compare native keys with index tie-break.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "array.slice",
      "identity": "can.intrinsic.array@1::slice",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "start",
          "type": "int"
        },
        {
          "name": "end",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.slice"
        ],
        "adapter": "Normalize slice bounds in bigint; return immutable native copies.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "array.concat",
      "identity": "can.intrinsic.array@1::concat",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "other",
          "type": "T[]"
        }
      ],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.concat"
        ],
        "adapter": "Normalize slice bounds in bigint; return immutable native copies.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "array.to_reversed",
      "identity": "can.intrinsic.array@1::to_reversed",
      "kind": "method",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.toReversed"
        ],
        "adapter": "Normalize slice bounds in bigint; return immutable native copies.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "append",
      "identity": "can.prelude@1::append",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "items",
          "type": "T[]"
        },
        {
          "name": "value",
          "type": "T"
        }
      ],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "array spread"
        ],
        "adapter": "Copy the source and append one value without mutation.",
        "task": "I21"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "str.includes",
      "identity": "can.intrinsic.str@1::includes",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.includes"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.starts_with",
      "identity": "can.intrinsic.str@1::starts_with",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.startsWith"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.ends_with",
      "identity": "can.intrinsic.str@1::ends_with",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.endsWith"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.to_lower_case",
      "identity": "can.intrinsic.str@1::to_lower_case",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.toLowerCase"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.to_upper_case",
      "identity": "can.intrinsic.str@1::to_upper_case",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.toUpperCase"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.trim",
      "identity": "can.intrinsic.str@1::trim",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.trim"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.slice",
      "identity": "can.intrinsic.str@1::slice",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [
        {
          "name": "start",
          "type": "int"
        },
        {
          "name": "end",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.slice"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.split",
      "identity": "can.intrinsic.str@1::split",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [
        {
          "name": "separator",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str[]",
      "callbacks": [],
      "emits": [
        "text::empty_separator"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.split"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "str.replace_all",
      "identity": "can.intrinsic.str@1::replace_all",
      "kind": "method",
      "receiver": "str",
      "parameters": [],
      "inputs": [
        {
          "name": "search",
          "type": "str"
        },
        {
          "name": "replacement",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "text::empty_pattern"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.replaceAll"
        ],
        "adapter": "Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "C7"
      ]
    },
    {
      "name": "text::join",
      "identity": "can.std.text@1::join",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "items",
          "type": "str[]"
        },
        {
          "name": "separator",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.join"
        ],
        "adapter": "Native join with explicit separator.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "text::scalars",
      "identity": "can.std.text@1::scalars",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "int[]",
      "callbacks": [],
      "emits": [
        "text::invalid_unicode"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.Symbol.iterator",
          "String.prototype.codePointAt"
        ],
        "adapter": "Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "text::from_scalars",
      "identity": "can.std.text@1::from_scalars",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "int[]"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "text::invalid_unicode"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.fromCodePoint"
        ],
        "adapter": "Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "text::graphemes",
      "identity": "can.std.text@1::graphemes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str[]",
      "callbacks": [],
      "emits": [
        "text::invalid_unicode"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Intl.Segmenter"
        ],
        "adapter": "Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "text::normalize_nfc",
      "identity": "can.std.text@1::normalize_nfc",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "text::invalid_unicode"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.normalize"
        ],
        "adapter": "Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization.",
        "task": "I24"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "text::compile_regex",
      "identity": "can.std.text@1::compile_regex",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "pattern",
          "type": "str"
        },
        {
          "name": "flags",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "text::regex",
      "callbacks": [],
      "emits": [
        "text::invalid_regex"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "RegExp"
        ],
        "adapter": "Compile validated patterns with i/m/s/u/v flags into opaque handles; other flags and bad patterns reject; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "text::matches",
      "identity": "can.std.text@1::matches",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "regex",
          "type": "text::regex"
        },
        {
          "name": "text",
          "type": "str"
        },
        {
          "name": "limit",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "text::regex_match[]",
      "callbacks": [],
      "emits": [
        "text::invalid_limit"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "RegExp"
        ],
        "adapter": "Scan with a fresh global pass per call (no shared lastIndex), UTF-16 offsets, empty-match advancement, absent captures as empty; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "collections::empty_map",
      "identity": "can.std.collections@1::empty_map",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "inputs": [],
      "staticInputs": [],
      "result": "collections::map<K,V>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::empty_set",
      "identity": "can.std.collections@1::empty_set",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "inputs": [],
      "staticInputs": [],
      "result": "collections::set<K>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::get",
      "identity": "can.std.collections@1::get",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "map",
          "type": "collections::map<K,V>"
        },
        {
          "name": "key",
          "type": "K"
        }
      ],
      "staticInputs": [],
      "result": "V",
      "callbacks": [],
      "emits": [
        "collections::key_absent"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map.prototype.has",
          "Map.prototype.get"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::insert",
      "identity": "can.std.collections@1::insert",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "map",
          "type": "collections::map<K,V>"
        },
        {
          "name": "key",
          "type": "K"
        },
        {
          "name": "value",
          "type": "V"
        }
      ],
      "staticInputs": [],
      "result": "collections::map<K,V>",
      "callbacks": [],
      "emits": [
        "collections::key_exists"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map",
          "Map.prototype.set"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::replace",
      "identity": "can.std.collections@1::replace",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "map",
          "type": "collections::map<K,V>"
        },
        {
          "name": "key",
          "type": "K"
        },
        {
          "name": "value",
          "type": "V"
        }
      ],
      "staticInputs": [],
      "result": "collections::map<K,V>",
      "callbacks": [],
      "emits": [
        "collections::key_absent"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map",
          "Map.prototype.set"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::remove",
      "identity": "can.std.collections@1::remove",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "map",
          "type": "collections::map<K,V>"
        },
        {
          "name": "key",
          "type": "K"
        }
      ],
      "staticInputs": [],
      "result": "collections::map<K,V>",
      "callbacks": [],
      "emits": [
        "collections::key_absent"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map",
          "Map.prototype.delete"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::entries",
      "identity": "can.std.collections@1::entries",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        },
        {
          "name": "V",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "map",
          "type": "collections::map<K,V>"
        }
      ],
      "staticInputs": [],
      "result": "collections::entry<K,V>[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map.prototype.entries",
          "Array.from"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::contains",
      "identity": "can.std.collections@1::contains",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "inputs": [
        {
          "name": "set",
          "type": "collections::set<K>"
        },
        {
          "name": "key",
          "type": "K"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set.prototype.has"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::add",
      "identity": "can.std.collections@1::add",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "inputs": [
        {
          "name": "set",
          "type": "collections::set<K>"
        },
        {
          "name": "key",
          "type": "K"
        }
      ],
      "staticInputs": [],
      "result": "collections::set<K>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set",
          "Set.prototype.add"
        ],
        "adapter": "Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::union",
      "identity": "can.std.collections@1::union",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "inputs": [
        {
          "name": "left",
          "type": "collections::set<K>"
        },
        {
          "name": "right",
          "type": "collections::set<K>"
        }
      ],
      "staticInputs": [],
      "result": "collections::set<K>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set.prototype.union"
        ],
        "adapter": "Intersection/difference filter the left iteration order; union retains left then unseen right keys.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::intersection",
      "identity": "can.std.collections@1::intersection",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "inputs": [
        {
          "name": "left",
          "type": "collections::set<K>"
        },
        {
          "name": "right",
          "type": "collections::set<K>"
        }
      ],
      "staticInputs": [],
      "result": "collections::set<K>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set",
          "Set.prototype.has",
          "Array.prototype.filter"
        ],
        "adapter": "Intersection/difference filter the left iteration order; union retains left then unseen right keys.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "collections::difference",
      "identity": "can.std.collections@1::difference",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "K",
          "constraint": "map_key"
        }
      ],
      "inputs": [
        {
          "name": "left",
          "type": "collections::set<K>"
        },
        {
          "name": "right",
          "type": "collections::set<K>"
        }
      ],
      "staticInputs": [],
      "result": "collections::set<K>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set",
          "Set.prototype.has",
          "Array.prototype.filter"
        ],
        "adapter": "Intersection/difference filter the left iteration order; union retains left then unseen right keys.",
        "task": "I25"
      },
      "assertion": "real",
      "refs": [
        "C7"
      ]
    },
    {
      "name": "number::divmod",
      "identity": "can.std.number@1::divmod",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "numerator",
          "type": "int"
        },
        {
          "name": "denominator",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "number::division",
      "callbacks": [],
      "emits": [
        "number::zero_divisor"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "bigint /",
          "bigint %",
          "bigint comparisons"
        ],
        "adapter": "Apply only C8 sign normalization and finite half-even remainder adjustment.",
        "task": "I23"
      },
      "assertion": "real",
      "refs": [
        "C8"
      ]
    },
    {
      "name": "number::euclidean_divmod",
      "identity": "can.std.number@1::euclidean_divmod",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "numerator",
          "type": "int"
        },
        {
          "name": "denominator",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "number::division",
      "callbacks": [],
      "emits": [
        "number::zero_divisor"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "bigint /",
          "bigint %",
          "bigint comparisons"
        ],
        "adapter": "Apply only C8 sign normalization and finite half-even remainder adjustment.",
        "task": "I23"
      },
      "assertion": "real",
      "refs": [
        "C8"
      ]
    },
    {
      "name": "number::round_ratio_half_even",
      "identity": "can.std.number@1::round_ratio_half_even",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "numerator",
          "type": "int"
        },
        {
          "name": "denominator",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "number::rounded",
      "callbacks": [],
      "emits": [
        "number::zero_divisor"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "bigint /",
          "bigint %",
          "bigint comparisons"
        ],
        "adapter": "Apply only C8 sign normalization and finite half-even remainder adjustment.",
        "task": "I23"
      },
      "assertion": "real",
      "refs": [
        "C8"
      ]
    },
    {
      "name": "bytes::empty",
      "identity": "can.std.bytes@1::empty",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Uint8Array"
        ],
        "adapter": "Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally.",
        "task": "I13"
      },
      "assertion": "real",
      "refs": [
        "A2"
      ]
    },
    {
      "name": "bytes::from_ints",
      "identity": "can.std.bytes@1::from_ints",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "values",
          "type": "int[]"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Uint8Array.from"
        ],
        "adapter": "Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally.",
        "task": "I13"
      },
      "assertion": "real",
      "refs": [
        "A2"
      ]
    },
    {
      "name": "bytes::to_ints",
      "identity": "can.std.bytes@1::to_ints",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "int[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.from"
        ],
        "adapter": "Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally.",
        "task": "I13"
      },
      "assertion": "real",
      "refs": [
        "A2"
      ]
    },
    {
      "name": "bytes::from_utf8",
      "identity": "can.std.bytes@1::from_utf8",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "TextEncoder"
        ],
        "adapter": "Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally.",
        "task": "I13"
      },
      "assertion": "real",
      "refs": [
        "A2"
      ]
    },
    {
      "name": "bytes::to_utf8",
      "identity": "can.std.bytes@1::to_utf8",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "TextDecoder"
        ],
        "adapter": "Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally.",
        "task": "I13"
      },
      "assertion": "real",
      "refs": [
        "A2"
      ]
    },
    {
      "name": "bytes::encode_base64",
      "identity": "can.std.bytes@1::encode_base64",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Buffer"
        ],
        "adapter": "Encode standard base64 with padding; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "bytes::decode_base64",
      "identity": "can.std.bytes@1::decode_base64",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "text",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Buffer"
        ],
        "adapter": "Decode standard base64 only after strict alphabet/padding gates; malformed text rejects, never truncates; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "bytes::encode_hex",
      "identity": "can.std.bytes@1::encode_hex",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Buffer"
        ],
        "adapter": "Encode lowercase hex; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "bytes::decode_hex",
      "identity": "can.std.bytes@1::decode_hex",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "text",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Buffer"
        ],
        "adapter": "Decode hex only after strict even-length alphabet gates; malformed text rejects, never truncates; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "codec::encode_json",
      "identity": "can.std.codec@1::encode_json",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "wire"
        }
      ],
      "inputs": [
        {
          "name": "value",
          "type": "T"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "JSON.parse",
          "JSON.rawJSON",
          "JSON.stringify",
          "TextEncoder",
          "TextDecoder"
        ],
        "adapter": "Derive nominal schema; preserve numeric tokens; guard duplicates, scalar text, cycles and A6 budgets.",
        "task": "I14"
      },
      "assertion": "real",
      "refs": [
        "A2",
        "A6"
      ]
    },
    {
      "name": "codec::decode_json",
      "identity": "can.std.codec@1::decode_json",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "wire"
        }
      ],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "T",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "JSON.parse",
          "JSON.rawJSON",
          "JSON.stringify",
          "TextEncoder",
          "TextDecoder"
        ],
        "adapter": "Derive nominal schema; preserve numeric tokens; guard duplicates, scalar text, cycles and A6 budgets.",
        "task": "I14"
      },
      "assertion": "real",
      "refs": [
        "A2",
        "A6"
      ]
    },
    {
      "name": "array.length",
      "identity": "can.intrinsic.array@1::length",
      "kind": "property",
      "receiver": "T[]",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.length",
          "BigInt"
        ],
        "adapter": "Exact native length widened to bigint; validate opaque bytes before access.",
        "task": "I07"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "A2"
      ]
    },
    {
      "name": "str.length",
      "identity": "can.intrinsic.str@1::length",
      "kind": "property",
      "receiver": "str",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.length",
          "BigInt"
        ],
        "adapter": "Exact native length widened to bigint; validate opaque bytes before access.",
        "task": "I07"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "A2"
      ]
    },
    {
      "name": "bytes.length",
      "identity": "can.intrinsic.bytes@1::length",
      "kind": "property",
      "receiver": "bytes::buffer",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Uint8Array.prototype.byteLength",
          "BigInt"
        ],
        "adapter": "Exact native length widened to bigint; validate opaque bytes before access.",
        "task": "I13"
      },
      "assertion": "real",
      "refs": [
        "C6",
        "A2"
      ]
    },
    {
      "name": "io::stdin_bytes",
      "identity": "can.std.io@1::stdin_bytes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "io::limit_exceeded",
        "io::read_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.stdin"
        ],
        "adapter": "Reject negative limits before input; count incrementally as bigint, cancel on overflow, preserve immutable bytes and fatal BOM-preserving UTF-8. Map only expected native I/O errors.",
        "task": "I29"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "io::stdin_text",
      "identity": "can.std.io@1::stdin_text",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "io::limit_exceeded",
        "io::read_failed",
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.stdin",
          "TextDecoder"
        ],
        "adapter": "Reject negative limits before input; count incrementally as bigint, cancel on overflow, preserve immutable bytes and fatal BOM-preserving UTF-8. Map only expected native I/O errors.",
        "task": "I29"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "io::stdout_write",
      "identity": "can.std.io@1::stdout_write",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "io::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.write",
          "Bun.stdout"
        ],
        "adapter": "Await write and return exact byte count.",
        "task": "I29"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "io::stderr_write",
      "identity": "can.std.io@1::stderr_write",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "io::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.write",
          "Bun.stderr"
        ],
        "adapter": "Await write and return exact byte count.",
        "task": "I29"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "clock::wall_millis",
      "identity": "can.std.clock@1::wall_millis",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Date.now",
          "BigInt"
        ],
        "adapter": "Return native epoch milliseconds exactly as bigint; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "clock::monotonic_millis",
      "identity": "can.std.clock@1::monotonic_millis",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "float",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "performance.now"
        ],
        "adapter": "Return native monotonic milliseconds as float; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "clock::sleep_millis",
      "identity": "can.std.clock@1::sleep_millis",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "milliseconds",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "clock::invalid_duration"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.sleep"
        ],
        "adapter": "Validate bigint milliseconds in 0--2147483647 before Number conversion and await Bun.sleep; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "random::secure_bytes",
      "identity": "can.std.random@1::secure_bytes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "length",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "random::invalid_length"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.getRandomValues",
          "Uint8Array"
        ],
        "adapter": "Validate bigint length in 0--65536 before allocation; fill a fresh native array and preserve immutable byte ownership; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "random::uuid_v4",
      "identity": "can.std.random@1::uuid_v4",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.randomUUID"
        ],
        "adapter": "Return native crypto.randomUUID; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "password::hash",
      "identity": "can.std.password@1::hash",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "password",
          "type": "str"
        },
        {
          "name": "profile",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "password::cost_rejected"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.password"
        ],
        "adapter": "Hash with async Bun.password (argon2id) under fixed preset 0..2 (fast 8MiB/1 pass, balanced 64MiB/2 passes, secure 256MiB/3 passes); other profiles reject; supplied assertion boundary.",
        "task": "B1-08"
      },
      "assertion": "supplied",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "password::verify",
      "identity": "can.std.password@1::verify",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "password",
          "type": "str"
        },
        {
          "name": "encoded",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [
        "password::invalid_hash"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.password"
        ],
        "adapter": "Gate the qualified argon2id envelope (version 19, memory 8 KiB..1 GiB, time 1..32, parallelism 1..4, 32-byte salt and hash) before native verify; malformed hashes reject while wrong passwords read false; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::sha256",
      "identity": "can.std.crypto@1::sha256",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "buffer",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.CryptoHasher"
        ],
        "adapter": "Hash copied immutable bytes with a fresh Bun.CryptoHasher and return owned digest bytes; execute in ordinary assertions.",
        "task": "I30"
      },
      "assertion": "real",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "crypto::hmac_sha256",
      "identity": "can.std.crypto@1::hmac_sha256",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "bytes::buffer"
        },
        {
          "name": "message",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "crypto::invalid_key"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Import the raw HMAC/SHA-256 key per call and sign; empty keys reject; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::generate_aes_key",
      "identity": "can.std.crypto@1::generate_aes_key",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "crypto::key",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Generate a native-nonextractable AES-256-GCM encrypt/decrypt handle; supplied assertion boundary.",
        "task": "B1-08"
      },
      "assertion": "supplied",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::generate_ed25519_keypair",
      "identity": "can.std.crypto@1::generate_ed25519_keypair",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "crypto::keypair",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Generate sign-only private plus verify-only public handles; supplied assertion boundary.",
        "task": "B1-08"
      },
      "assertion": "supplied",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::import_ed25519_public",
      "identity": "can.std.crypto@1::import_ed25519_public",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "public",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "crypto::key",
      "callbacks": [],
      "emits": [
        "crypto::invalid_key"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Admit 32-byte verify-only Ed25519 public keys; other lengths reject; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::export_ed25519_public",
      "identity": "can.std.crypto@1::export_ed25519_public",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "crypto::key"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "crypto::key_misuse"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Export raw bytes only from verify-only Ed25519 handles; sign-capable and AES handles misuse; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::encrypt_aes_gcm",
      "identity": "can.std.crypto@1::encrypt_aes_gcm",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "crypto::key"
        },
        {
          "name": "nonce",
          "type": "bytes::buffer"
        },
        {
          "name": "plaintext",
          "type": "bytes::buffer"
        },
        {
          "name": "associated_data",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "crypto::invalid_nonce",
        "crypto::key_misuse"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Encrypt with a fixed 12-byte nonce and 128-bit tag through usage-checked handles; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::encrypt_aes_gcm_sealed",
      "identity": "can.std.crypto@1::encrypt_aes_gcm_sealed",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "crypto::key"
        },
        {
          "name": "plaintext",
          "type": "bytes::buffer"
        },
        {
          "name": "associated_data",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "crypto::sealed",
      "callbacks": [],
      "emits": [
        "crypto::key_misuse"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Mint a native-random 12-byte nonce and return it with the ciphertext; generation is randomness, not a guarantee of caller nonce discipline; supplied assertion boundary.",
        "task": "B1-08"
      },
      "assertion": "supplied",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::decrypt_aes_gcm",
      "identity": "can.std.crypto@1::decrypt_aes_gcm",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "crypto::key"
        },
        {
          "name": "nonce",
          "type": "bytes::buffer"
        },
        {
          "name": "ciphertext",
          "type": "bytes::buffer"
        },
        {
          "name": "associated_data",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "crypto::invalid_nonce",
        "crypto::key_misuse",
        "crypto::decrypt_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Decrypt with a fixed 12-byte nonce and 128-bit tag; tampering, wrong keys, and associated-data mismatch collapse to decrypt_failed; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::sign_ed25519",
      "identity": "can.std.crypto@1::sign_ed25519",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "crypto::key"
        },
        {
          "name": "message",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "crypto::key_misuse"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Sign through sign-capable handles; deterministic 64-byte signatures; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "crypto::verify_ed25519",
      "identity": "can.std.crypto@1::verify_ed25519",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "key",
          "type": "crypto::key"
        },
        {
          "name": "message",
          "type": "bytes::buffer"
        },
        {
          "name": "signature",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [
        "crypto::key_misuse"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "crypto.subtle"
        ],
        "adapter": "Verify through verify-capable handles; mismatch reads false; execute in ordinary assertions.",
        "task": "B1-08"
      },
      "assertion": "real",
      "refs": [
        "B1-08"
      ]
    },
    {
      "name": "env::required",
      "identity": "can.std.env@1::required",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "env::invalid_name",
        "http::credentials_missing"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.env"
        ],
        "adapter": "Validate uppercase environment names before exact caller-snapshot lookup. Absence is distinct from a present empty string; assertion execution requires supplied completions.",
        "task": "I29"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "env::optional",
      "identity": "can.std.env@1::optional",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "option::value<str>",
      "callbacks": [],
      "emits": [
        "env::invalid_name"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.env"
        ],
        "adapter": "Validate uppercase environment names before exact caller-snapshot lookup. Absence is distinct from a present empty string; assertion execution requires supplied completions.",
        "task": "I29"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "log::write_info",
      "identity": "can.std.log@1::write_info",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "message",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "log::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "JSON.stringify",
          "Bun.write"
        ],
        "adapter": "Serialize exactly level/info and message string fields with native JSON.stringify, append newline, await stderr; map serialization and expected I/O failures to a level-only payload; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "log::write_error",
      "identity": "can.std.log@1::write_error",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "message",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "log::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "JSON.stringify",
          "Bun.write"
        ],
        "adapter": "Serialize exactly level/error and message string fields with native JSON.stringify, append newline, await stderr; map serialization and expected I/O failures to a level-only payload; supplied assertion boundary.",
        "task": "I30"
      },
      "assertion": "supplied",
      "refs": [
        "P8"
      ]
    },
    {
      "name": "html::make_tag",
      "identity": "can.std.html@1::make_tag",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "html::tag",
      "callbacks": [],
      "emits": [
        "html::invalid_structure"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Set.prototype.has"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::text",
      "identity": "can.std.html@1::text",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "html::node",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::parse_url",
      "identity": "can.std.html@1::parse_url",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "html::url",
      "callbacks": [],
      "emits": [
        "html::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::text_attribute",
      "identity": "can.std.html@1::text_attribute",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "html::invalid_structure"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::url_attribute",
      "identity": "can.std.html@1::url_attribute",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "url",
          "type": "html::url"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "html::invalid_structure"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::element",
      "identity": "can.std.html@1::element",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "tag",
          "type": "html::tag"
        },
        {
          "name": "attributes",
          "type": "html::attribute[]"
        },
        {
          "name": "children",
          "type": "html::node[]"
        }
      ],
      "staticInputs": [],
      "result": "html::node",
      "callbacks": [],
      "emits": [
        "html::invalid_structure"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.join",
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::fragment",
      "identity": "can.std.html@1::fragment",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "nodes",
          "type": "html::node[]"
        }
      ],
      "staticInputs": [],
      "result": "html::safe",
      "callbacks": [],
      "emits": [
        "html::invalid_structure"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Array.prototype.join"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::text_fragment",
      "identity": "can.std.html@1::text_fragment",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "html::safe",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::stylesheet",
      "identity": "can.std.html@1::stylesheet",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "html::url"
        }
      ],
      "staticInputs": [],
      "result": "html::node",
      "callbacks": [],
      "emits": [
        "html::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::meta_viewport",
      "identity": "can.std.html@1::meta_viewport",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "html::node",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "string literal"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "html::document",
      "identity": "can.std.html@1::document",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "title",
          "type": "str"
        },
        {
          "name": "head",
          "type": "html::node[]"
        },
        {
          "name": "body",
          "type": "html::node[]"
        }
      ],
      "staticInputs": [],
      "result": "html::safe",
      "callbacks": [],
      "emits": [
        "html::invalid_structure"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML",
          "Array.prototype.join"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "htmx::get",
      "identity": "can.std.htmx@1::get",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "html::url"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "html::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "htmx::post",
      "identity": "can.std.htmx@1::post",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "html::url"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "html::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "htmx::target_id",
      "identity": "can.std.htmx@1::target_id",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "id",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "htmx::target",
      "callbacks": [],
      "emits": [
        "htmx::invalid_target"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String.prototype.match"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "htmx::target_attribute",
      "identity": "can.std.htmx@1::target_attribute",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "target",
          "type": "htmx::target"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "htmx::indicator_id",
      "identity": "can.std.htmx@1::indicator_id",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "id",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "htmx::invalid_target"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.escapeHTML"
        ],
        "adapter": "Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P6",
        "P9"
      ]
    },
    {
      "name": "htmx::swap_inner",
      "identity": "can.std.htmx@1::swap_inner",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "string literal"
        ],
        "adapter": "Construct only the fixed P9 HTMX attribute.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P9"
      ]
    },
    {
      "name": "htmx::swap_outer",
      "identity": "can.std.htmx@1::swap_outer",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "string literal"
        ],
        "adapter": "Construct only the fixed P9 HTMX attribute.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P9"
      ]
    },
    {
      "name": "htmx::trigger_change",
      "identity": "can.std.htmx@1::trigger_change",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "string literal"
        ],
        "adapter": "Construct only the fixed P9 HTMX attribute.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P9"
      ]
    },
    {
      "name": "htmx::disable_this",
      "identity": "can.std.htmx@1::disable_this",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "string literal"
        ],
        "adapter": "Construct only the fixed P9 HTMX attribute.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P9"
      ]
    },
    {
      "name": "htmx::trigger_input_changed",
      "identity": "can.std.htmx@1::trigger_input_changed",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "delay_ms",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "htmx::invalid_interval"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String"
        ],
        "adapter": "Validate P9 interval bounds and construct the fixed trigger.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P9"
      ]
    },
    {
      "name": "htmx::trigger_every",
      "identity": "can.std.htmx@1::trigger_every",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "interval_ms",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "html::attribute",
      "callbacks": [],
      "emits": [
        "htmx::invalid_interval"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "String"
        ],
        "adapter": "Validate P9 interval bounds and construct the fixed trigger.",
        "task": "I31"
      },
      "assertion": "real",
      "refs": [
        "P9"
      ]
    },
    {
      "name": "htmx::runtime_head",
      "identity": "can.std.htmx@1::runtime_head",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "html::node",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "string literal"
        ],
        "adapter": "Emit only the pinned P11 local HTMX script and fixed response/config policy.",
        "task": "I34"
      },
      "assertion": "real",
      "refs": [
        "P9",
        "P11"
      ]
    },
    {
      "name": "asset::url",
      "identity": "can.std.asset@1::url",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [
        "name"
      ],
      "result": "html::url",
      "callbacks": [],
      "emits": [
        "html::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Resolve a static declared asset to its validated build-manifest URL.",
        "task": "I34"
      },
      "assertion": "real",
      "refs": [
        "P11"
      ]
    },
    {
      "name": "http::request_method",
      "identity": "can.std.http@1::request_method",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Request",
          "URL",
          "Headers"
        ],
        "adapter": "Read the immutable request snapshot and expose copied Can data.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_path",
      "identity": "can.std.http@1::request_path",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Request",
          "URL",
          "Headers"
        ],
        "adapter": "Read the immutable request snapshot and expose copied Can data.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_headers",
      "identity": "can.std.http@1::request_headers",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        }
      ],
      "staticInputs": [],
      "result": "http::header[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Request",
          "URL",
          "Headers"
        ],
        "adapter": "Read the immutable request snapshot and expose copied Can data.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::query_one",
      "identity": "can.std.http@1::query_one",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URLSearchParams"
        ],
        "adapter": "Check missing/repeated single query values; preserve repeated order.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::query_all",
      "identity": "can.std.http@1::query_all",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str[]",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URLSearchParams"
        ],
        "adapter": "Check missing/repeated single query values; preserve repeated order.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_body",
      "identity": "can.std.http@1::request_body",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "http::body_limit"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Request",
          "URLSearchParams",
          "JSON.parse",
          "TextDecoder"
        ],
        "adapter": "Read cached bounded bytes; apply the declared media/percent/UTF-8/typed codec policy.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_json",
      "identity": "can.std.http@1::request_json",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "wire"
        }
      ],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "T",
      "callbacks": [],
      "emits": [
        "http::body_limit",
        "http::invalid_request",
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Request",
          "URLSearchParams",
          "JSON.parse",
          "TextDecoder"
        ],
        "adapter": "Read cached bounded bytes; apply the declared media/percent/UTF-8/typed codec policy.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_form",
      "identity": "can.std.http@1::request_form",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "form"
        }
      ],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "T",
      "callbacks": [],
      "emits": [
        "http::body_limit",
        "http::invalid_request",
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Request",
          "URLSearchParams",
          "JSON.parse",
          "TextDecoder"
        ],
        "adapter": "Read cached bounded bytes; apply the declared media/percent/UTF-8/typed codec policy.",
        "task": "I32"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_multipart",
      "identity": "can.std.http@1::request_multipart",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "max_bytes",
          "type": "int"
        },
        {
          "name": "max_file_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "http::multipart_form",
      "callbacks": [],
      "emits": [
        "http::body_limit",
        "http::invalid_request",
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "TextDecoder"
        ],
        "adapter": "Parse bounded flat multipart into generic field/file records.",
        "task": "B1-06"
      },
      "assertion": "scoped",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::make_status",
      "identity": "can.std.http@1::make_status",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "http::status",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Admit 200\u2013599; body status additionally excludes 204, 205 and 304.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::make_body_status",
      "identity": "can.std.http@1::make_body_status",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "http::body_status",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Admit 200\u2013599; body status additionally excludes 204, 205 and 304.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::status_ok",
      "identity": "can.std.http@1::status_ok",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "http::body_status",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Construct the corresponding fixed validated status.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::status_unprocessable",
      "identity": "can.std.http@1::status_unprocessable",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "http::body_status",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Construct the corresponding fixed validated status.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::status_internal",
      "identity": "can.std.http@1::status_internal",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "http::body_status",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Construct the corresponding fixed validated status.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::status_unavailable",
      "identity": "can.std.http@1::status_unavailable",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "http::body_status",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Construct the corresponding fixed validated status.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::make_server_headers",
      "identity": "can.std.http@1::make_server_headers",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "headers",
          "type": "http::header[]"
        }
      ],
      "staticInputs": [],
      "result": "http::server_headers",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Headers"
        ],
        "adapter": "Validate names/values and reject forbidden content/hop-by-hop fields.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::empty_server_headers",
      "identity": "can.std.http@1::empty_server_headers",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "http::server_headers",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Headers"
        ],
        "adapter": "Create the empty immutable header set.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_empty",
      "identity": "can.std.http@1::response_empty",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "http::status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Response",
          "Headers",
          "TextEncoder"
        ],
        "adapter": "Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_bytes",
      "identity": "can.std.http@1::response_bytes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "http::body_status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        },
        {
          "name": "body",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Response",
          "Headers",
          "TextEncoder"
        ],
        "adapter": "Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_text",
      "identity": "can.std.http@1::response_text",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "http::body_status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        },
        {
          "name": "body",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Response",
          "Headers",
          "TextEncoder"
        ],
        "adapter": "Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_html",
      "identity": "can.std.http@1::response_html",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "http::body_status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        },
        {
          "name": "body",
          "type": "html::safe"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Response",
          "Headers",
          "TextEncoder"
        ],
        "adapter": "Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_json",
      "identity": "can.std.http@1::response_json",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "wire"
        }
      ],
      "inputs": [
        {
          "name": "status",
          "type": "http::body_status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        },
        {
          "name": "body",
          "type": "T"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [
        "codec::invalid_data"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Response",
          "Headers",
          "JSON.stringify"
        ],
        "adapter": "Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_stream",
      "identity": "can.std.http@1::response_stream",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "http::body_status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStream"
        ],
        "adapter": "Build a pending response whose bounded queue the vended writer fills.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_writer",
      "identity": "can.std.http@1::response_writer",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "response",
          "type": "http::server_response"
        }
      ],
      "staticInputs": [],
      "result": "stream::writer",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStream"
        ],
        "adapter": "Vend the pending response writer exactly once for short-write production.",
        "task": "B1-06"
      },
      "assertion": "supplied",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::response_sse",
      "identity": "can.std.http@1::response_sse",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "status",
          "type": "http::body_status"
        },
        {
          "name": "headers",
          "type": "http::server_headers"
        }
      ],
      "staticInputs": [],
      "result": "http::server_response",
      "callbacks": [],
      "emits": [
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStream"
        ],
        "adapter": "Build a pending event-stream response for validated SSE production.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::sse_send",
      "identity": "can.std.http@1::sse_send",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "writer",
          "type": "stream::writer"
        },
        {
          "name": "event",
          "type": "http::sse_event"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "http::invalid_request",
        "http::body_limit",
        "stream::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStream",
          "TextEncoder"
        ],
        "adapter": "Validate one event frame and append it atomically to the queue.",
        "task": "B1-06"
      },
      "assertion": "supplied",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::sse_comment",
      "identity": "can.std.http@1::sse_comment",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "writer",
          "type": "stream::writer"
        },
        {
          "name": "text",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "http::invalid_request",
        "http::body_limit",
        "stream::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStream",
          "TextEncoder"
        ],
        "adapter": "Validate one comment line and append it atomically to the queue.",
        "task": "B1-06"
      },
      "assertion": "supplied",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_get",
      "identity": "can.std.http@1::route_get",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_post",
      "identity": "can.std.http@1::route_post",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_put",
      "identity": "can.std.http@1::route_put",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_patch",
      "identity": "can.std.http@1::route_patch",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_delete",
      "identity": "can.std.http@1::route_delete",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_options",
      "identity": "can.std.http@1::route_options",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_head",
      "identity": "can.std.http@1::route_head",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [
        "path"
      ],
      "result": "http::route",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "http::request"
          ],
          "result": "http::server_response",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "http::invalid_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Validate exact normalized path and mount a named boxed Can callback.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::make_router",
      "identity": "can.std.http@1::make_router",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "routes",
          "type": "http::route[]"
        }
      ],
      "staticInputs": [],
      "result": "http::router",
      "callbacks": [],
      "emits": [
        "http::duplicate_route",
        "http::ambiguous_route"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map"
        ],
        "adapter": "Closed exact dispatch with 404/405/Allow, no implicit HEAD.",
        "task": "I32"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::route_stream",
      "identity": "can.std.http@1::route_stream",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "route",
          "type": "http::route"
        }
      ],
      "staticInputs": [],
      "result": "http::route",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Map"
        ],
        "adapter": "Mark a route for lazy bodies consumed once through a stream reader.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::request_body_stream",
      "identity": "can.std.http@1::request_body_stream",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "max_chunk",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "stream::reader<bytes::buffer>",
      "callbacks": [],
      "emits": [
        "http::body_limit",
        "http::invalid_request"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStream"
        ],
        "adapter": "Open the one-shot body reader: live wire bytes or replayed buffered bytes.",
        "task": "B1-06"
      },
      "assertion": "supplied",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::make_server_config",
      "identity": "can.std.http@1::make_server_config",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "host",
          "type": "str"
        },
        {
          "name": "port",
          "type": "int"
        },
        {
          "name": "body_limit",
          "type": "int"
        },
        {
          "name": "shutdown_ms",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "http::server_config",
      "callbacks": [],
      "emits": [
        "http::invalid_server_config"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Number"
        ],
        "adapter": "Validate bounded config before server start.",
        "task": "I33"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::server_start",
      "identity": "can.std.http@1::server_start",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "config",
          "type": "http::server_config"
        },
        {
          "name": "router",
          "type": "http::router"
        }
      ],
      "staticInputs": [],
      "result": "http::server",
      "callbacks": [],
      "emits": [
        "http::bind_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.serve"
        ],
        "adapter": "Register ownership; await each Can callback and sanitize standard failures.",
        "task": "I33"
      },
      "assertion": "supplied",
      "refs": [
        "P6",
        "P10"
      ]
    },
    {
      "name": "http::make_tls_config",
      "identity": "can.std.http@1::make_tls_config",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "cert",
          "type": "bytes::buffer"
        },
        {
          "name": "key",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "http::tls_config",
      "callbacks": [],
      "emits": [
        "http::invalid_server_config"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "TextDecoder"
        ],
        "adapter": "Validate PEM certificate chain and private key structure before server start.",
        "task": "B1-06"
      },
      "assertion": "real",
      "refs": [
        "P10"
      ]
    },
    {
      "name": "http::server_start_tls",
      "identity": "can.std.http@1::server_start_tls",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "config",
          "type": "http::server_config"
        },
        {
          "name": "router",
          "type": "http::router"
        },
        {
          "name": "tls",
          "type": "http::tls_config"
        }
      ],
      "staticInputs": [],
      "result": "http::server",
      "callbacks": [],
      "emits": [
        "http::bind_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.serve"
        ],
        "adapter": "Register ownership; serve TLS with the validated material and sanitize standard failures.",
        "task": "B1-06"
      },
      "assertion": "supplied",
      "refs": [
        "P6",
        "P10"
      ]
    },
    {
      "name": "http::server_wait",
      "identity": "can.std.http@1::server_wait",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "server",
          "type": "http::server"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "http::shutdown_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.Server.stop",
          "process.on"
        ],
        "adapter": "Wait for graceful stop(false), native stop and leases; timeout does not revoke ownership.",
        "task": "I33"
      },
      "assertion": "supplied",
      "refs": [
        "P6",
        "P10"
      ]
    },
    {
      "name": "http::server_stop",
      "identity": "can.std.http@1::server_stop",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "server",
          "type": "http::server"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "http::shutdown_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.Server.stop",
          "process.on"
        ],
        "adapter": "Wait for graceful stop(false), native stop and leases; timeout does not revoke ownership.",
        "task": "I33"
      },
      "assertion": "supplied",
      "refs": [
        "P6",
        "P10"
      ]
    },
    {
      "name": "sql::pool_open",
      "identity": "can.std.sql@1::pool_open",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "connection_variable",
          "type": "str"
        },
        {
          "name": "max_connections",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "sql::pool",
      "callbacks": [],
      "emits": [
        "http::credentials_missing",
        "sql::connection_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL"
        ],
        "adapter": "Read selected credential; open PostgreSQL with bigint:true and validated max.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::pool_close",
      "identity": "can.std.sql@1::pool_close",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "pool",
          "type": "sql::pool"
        },
        {
          "name": "timeout_ms",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "sql::close_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL.close"
        ],
        "adapter": "Drain leases then close with remaining deadline; leave timed-out close owned.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P6",
        "P12"
      ]
    },
    {
      "name": "sql::sqlite_open_memory",
      "identity": "can.std.sql@1::sqlite_open_memory",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [],
      "staticInputs": [],
      "result": "sql::pool",
      "callbacks": [],
      "emits": [
        "sql::connection_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL"
        ],
        "adapter": "Open in-memory SQLite with safeIntegers:true; the database lives while the pool is open.",
        "task": "B1-02"
      },
      "assertion": "supplied",
      "refs": [
        "B1-02"
      ]
    },
    {
      "name": "sql::sqlite_open_file",
      "identity": "can.std.sql@1::sqlite_open_file",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "options",
          "type": "sql::sqlite_file_options"
        }
      ],
      "staticInputs": [],
      "result": "sql::pool",
      "callbacks": [],
      "emits": [
        "sql::connection_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL"
        ],
        "adapter": "Open file SQLite with safeIntegers:true; mode is ro, rw or rwc and busy_timeout_ms bounds lock waits.",
        "task": "B1-02"
      },
      "assertion": "supplied",
      "refs": [
        "B1-02"
      ]
    },
    {
      "name": "sql::mysql_open",
      "identity": "can.std.sql@1::mysql_open",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "connection_variable",
          "type": "str"
        },
        {
          "name": "max_connections",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "sql::pool",
      "callbacks": [],
      "emits": [
        "http::credentials_missing",
        "sql::connection_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL"
        ],
        "adapter": "Read selected credential; open MySQL with bigint:true, forced TLS, validated max, and a pinned UTC session.",
        "task": "B1-03"
      },
      "assertion": "supplied",
      "refs": [
        "B1-03"
      ]
    },
    {
      "name": "sql::query_one",
      "identity": "can.std.sql@1::query_one",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        },
        {
          "name": "R",
          "constraint": "sql_row"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::pool"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "R",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::connection_failed",
        "sql::query_failed",
        "sql::constraint_failed",
        "sql::row_missing",
        "sql::row_count",
        "sql::schema_mismatch"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::query_optional",
      "identity": "can.std.sql@1::query_optional",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        },
        {
          "name": "R",
          "constraint": "sql_row"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::pool"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "option::value<R>",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::connection_failed",
        "sql::query_failed",
        "sql::constraint_failed",
        "sql::row_count",
        "sql::schema_mismatch"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::query_rows",
      "identity": "can.std.sql@1::query_rows",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        },
        {
          "name": "R",
          "constraint": "sql_row"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::pool"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        },
        {
          "name": "max_rows",
          "type": "int"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "R[]",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::connection_failed",
        "sql::query_failed",
        "sql::constraint_failed",
        "sql::row_limit",
        "sql::schema_mismatch"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::execute",
      "identity": "can.std.sql@1::execute",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::pool"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "int",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::connection_failed",
        "sql::query_failed",
        "sql::constraint_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::transaction_query_one",
      "identity": "can.std.sql@1::transaction_query_one",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        },
        {
          "name": "R",
          "constraint": "sql_row"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::transaction"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "R",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::query_failed",
        "sql::constraint_failed",
        "sql::row_missing",
        "sql::row_count",
        "sql::schema_mismatch"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::transaction_query_optional",
      "identity": "can.std.sql@1::transaction_query_optional",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        },
        {
          "name": "R",
          "constraint": "sql_row"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::transaction"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "option::value<R>",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::query_failed",
        "sql::constraint_failed",
        "sql::row_count",
        "sql::schema_mismatch"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::transaction_query_rows",
      "identity": "can.std.sql@1::transaction_query_rows",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        },
        {
          "name": "R",
          "constraint": "sql_row"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::transaction"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        },
        {
          "name": "max_rows",
          "type": "int"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "R[]",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::query_failed",
        "sql::constraint_failed",
        "sql::row_limit",
        "sql::schema_mismatch"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::transaction_execute",
      "identity": "can.std.sql@1::transaction_execute",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "P",
          "constraint": "sql_parameters"
        }
      ],
      "inputs": [
        {
          "name": "handle",
          "type": "sql::transaction"
        },
        {
          "name": "descriptor",
          "type": "str"
        },
        {
          "name": "parameters",
          "type": "P"
        }
      ],
      "staticInputs": [
        "descriptor"
      ],
      "result": "int",
      "callbacks": [],
      "emits": [
        "sql::unsupported_value",
        "sql::query_failed",
        "sql::constraint_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL tagged template"
        ],
        "adapter": "Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1.",
        "task": "I35"
      },
      "assertion": "supplied",
      "refs": [
        "P12"
      ]
    },
    {
      "name": "sql::with_transaction",
      "identity": "can.std.sql@1::with_transaction",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "pool",
          "type": "sql::pool"
        },
        {
          "name": "callback",
          "type": "$callback"
        }
      ],
      "staticInputs": [],
      "result": "T",
      "callbacks": [
        {
          "name": "callback",
          "inputs": [
            "sql::transaction"
          ],
          "result": "sql::decision<T>",
          "deriveErrors": false,
          "emits": []
        }
      ],
      "emits": [
        "sql::connection_failed",
        "sql::transaction_failed",
        "sql::commit_unknown"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.SQL.begin"
        ],
        "adapter": "Drain scoped leases; private rollback sentinel; retain commit uncertainty and original standard failures.",
        "task": "I38"
      },
      "assertion": "scoped",
      "refs": [
        "P6",
        "P12"
      ]
    },
    {
      "name": "checks::require",
      "identity": "can.std.checks@1::require",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "condition",
          "type": "bool"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "checks::failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Boolean branch",
          "domain.create"
        ],
        "adapter": "Evaluate condition then reason once each; false produces checks::failed with the exact authored reason. Record the call-site span and invocation path in private occurrence metadata.",
        "task": "LF08"
      },
      "assertion": "real",
      "refs": [
        "C9.2"
      ]
    },
    {
      "name": "files::read_bytes",
      "identity": "can.std.files@1::read_bytes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "bytes::buffer",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::limit_exceeded",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.file"
        ],
        "adapter": "Reject empty/NUL paths and negative limits before input; stream Bun.file chunks counting bigint bytes before retaining, copy each native view, cancel and release on overflow; map EISDIR to unexpected_kind; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::read_text",
      "identity": "can.std.files@1::read_text",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "max_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::limit_exceeded",
        "codec::invalid_data",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.file",
          "TextDecoder"
        ],
        "adapter": "Bounded read_bytes then fatal UTF-8 decode; undecodable input is codec::invalid_data with the read path; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::write_bytes",
      "identity": "can.std.files@1::write_bytes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "value",
          "type": "bytes::buffer"
        },
        {
          "name": "overwrite",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::already_exists",
        "files::denied",
        "files::invalid_path",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.writeFile"
        ],
        "adapter": "Copy Can bytes out; write with flag wx when overwrite is false so exclusive creation is atomic; never auto-create missing parents; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::write_text",
      "identity": "can.std.files@1::write_text",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        },
        {
          "name": "overwrite",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::already_exists",
        "files::denied",
        "files::invalid_path",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "TextEncoder",
          "node:fs/promises.writeFile"
        ],
        "adapter": "Encode UTF-8 then the write_bytes contract; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::stat",
      "identity": "can.std.files@1::stat",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "follow_symlinks",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "files::file_info",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.stat",
          "node:fs/promises.lstat"
        ],
        "adapter": "Use stat when following and lstat otherwise; project kind file/directory/symlink/other with exact bigint size; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::exists",
      "identity": "can.std.files@1::exists",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [
        "files::denied",
        "files::invalid_path",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.lstat"
        ],
        "adapter": "Report true for any entry kind including dangling symlinks; only missing paths report false, never permission failures; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::list",
      "identity": "can.std.files@1::list",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "max_entries",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "files::entry[]",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::limit_exceeded",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.readdir",
          "node:path.resolve",
          "node:path.join"
        ],
        "adapter": "Read typed entries once, resolve each child to an absolute path, sort lexically, and reject over-limit directories instead of truncating; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::mkdir",
      "identity": "can.std.files@1::mkdir",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "recursive",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::already_exists",
        "files::denied",
        "files::invalid_path",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.mkdir"
        ],
        "adapter": "Create one directory or a recursive chain; an existing path fails only when recursive is false; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::copy",
      "identity": "can.std.files@1::copy",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "source",
          "type": "str"
        },
        {
          "name": "destination",
          "type": "str"
        },
        {
          "name": "overwrite",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::already_exists",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.copyFile"
        ],
        "adapter": "Copy bytes with COPYFILE_EXCL unless overwrite; attribute missing-path failures to the absent side best-effort; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::move",
      "identity": "can.std.files@1::move",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "source",
          "type": "str"
        },
        {
          "name": "destination",
          "type": "str"
        },
        {
          "name": "overwrite",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::already_exists",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::not_empty",
        "files::cross_device",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.rename"
        ],
        "adapter": "Rename without copy fallback; cross-device moves fail explicitly and never silently lose atomicity; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::remove",
      "identity": "can.std.files@1::remove",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "recursive",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::not_empty",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:fs/promises.rm",
          "node:fs/promises.rmdir",
          "node:fs/promises.unlink"
        ],
        "adapter": "Remove one file/symlink/empty directory, or a recursive tree only when requested; a non-empty directory without recursion is not_empty; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "files::glob",
      "identity": "can.std.files@1::glob",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "base",
          "type": "str"
        },
        {
          "name": "pattern",
          "type": "str"
        },
        {
          "name": "follow_symlinks",
          "type": "bool"
        },
        {
          "name": "max_entries",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str[]",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::limit_exceeded",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.Glob"
        ],
        "adapter": "Enumerate natively with the entry cap enforced during iteration, return absolute sorted paths including directories, and never silently truncate; supplied assertion boundary.",
        "task": "B1-01"
      },
      "assertion": "supplied",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "path::resolve",
      "identity": "can.std.path@1::resolve",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "base",
          "type": "str"
        },
        {
          "name": "parts",
          "type": "str[]"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:path.resolve"
        ],
        "adapter": "Resolve parts against the base with native normalization; pure computation.",
        "task": "B1-01"
      },
      "assertion": "real",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "path::join",
      "identity": "can.std.path@1::join",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "parts",
          "type": "str[]"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:path.join"
        ],
        "adapter": "Join segments with native normalization; pure computation.",
        "task": "B1-01"
      },
      "assertion": "real",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "path::basename",
      "identity": "can.std.path@1::basename",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:path.basename"
        ],
        "adapter": "Return the final segment natively; pure computation.",
        "task": "B1-01"
      },
      "assertion": "real",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "path::extension",
      "identity": "can.std.path@1::extension",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "node:path.extname"
        ],
        "adapter": "Return the native extension including the leading dot, or empty; pure computation.",
        "task": "B1-01"
      },
      "assertion": "real",
      "refs": [
        "B1-01"
      ]
    },
    {
      "name": "process::run",
      "identity": "can.std.process@1::run",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "executable",
          "type": "str"
        },
        {
          "name": "args",
          "type": "str[]"
        },
        {
          "name": "options",
          "type": "process::options"
        }
      ],
      "staticInputs": [],
      "result": "process::result",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "process::spawn_failed",
        "process::timeout",
        "process::output_limit",
        "process::invalid_config",
        "process::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.spawn"
        ],
        "adapter": "Spawn detached in its own process group with piped stdio, no shell; drain both streams concurrently under caps, enforce the deadline, and terminate the group SIGTERM-then-SIGKILL with a grace before escalation; reap every child and register the run as an owned resource so scope drain kills survivors; supplied assertion boundary.",
        "task": "B1-04"
      },
      "assertion": "supplied",
      "refs": [
        "B1-04"
      ]
    },
    {
      "name": "process::require_success",
      "identity": "can.std.process@1::require_success",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "value",
          "type": "process::result"
        }
      ],
      "staticInputs": [],
      "result": "process::result",
      "callbacks": [],
      "emits": [
        "process::nonzero"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "domain.create"
        ],
        "adapter": "Return the result unchanged when it exited zero, else nonzero with the observed code and signal; pure computation.",
        "task": "B1-04"
      },
      "assertion": "real",
      "refs": [
        "B1-04"
      ]
    },
    {
      "name": "process::which",
      "identity": "can.std.process@1::which",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "process::invalid_config"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.which"
        ],
        "adapter": "Resolve the executable natively; an unresolvable name is files::not_found and an empty name is invalid_config; supplied assertion boundary.",
        "task": "B1-04"
      },
      "assertion": "supplied",
      "refs": [
        "B1-04"
      ]
    },
    {
      "name": "stream::read_many",
      "identity": "can.std.stream@1::read_many",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "reader",
          "type": "stream::reader<T>"
        },
        {
          "name": "max_items",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "T[]",
      "callbacks": [],
      "emits": [
        "stream::read_failed",
        "stream::cancelled",
        "files::limit_exceeded"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStreamDefaultReader.read"
        ],
        "adapter": "One native pull per batch step with no prefetch queue; empty batch is the normal end; bytes items split at max_chunk with copied views, text items \\n-framed with fatal UTF-8; interrupted reads report cancelled and deliver nothing; failures are terminal; use-after-close/foreign-owner throw standard resource-state.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "stream::write_some",
      "identity": "can.std.stream@1::write_some",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "writer",
          "type": "stream::writer"
        },
        {
          "name": "chunk",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "stream::write_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "FileSink.write"
        ],
        "adapter": "Copy payload bytes out of immutable values, report accepted count, leave short-write retries to the caller; no coalescing queue.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "stream::close_reader",
      "identity": "can.std.stream@1::close_reader",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "reader",
          "type": "stream::reader<T>"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "stream::close_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStreamDefaultReader.cancel"
        ],
        "adapter": "Terminal owner close with bounded shutdown; cancels the native reader and releases the lock exactly once; close twice throws standard resource-state.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "stream::close_writer",
      "identity": "can.std.stream@1::close_writer",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "writer",
          "type": "stream::writer"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "stream::close_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "FileSink.end"
        ],
        "adapter": "Terminal owner close with bounded shutdown; flushes and ends the sink exactly once; close twice throws standard resource-state.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "stream::cancel_reader",
      "identity": "can.std.stream@1::cancel_reader",
      "kind": "function",
      "receiver": "",
      "parameters": [
        {
          "name": "T",
          "constraint": "data"
        }
      ],
      "inputs": [
        {
          "name": "reader",
          "type": "stream::reader<T>"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "stream::close_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "ReadableStreamDefaultReader.cancel"
        ],
        "adapter": "Record the reason, then terminal owner close; an in-flight read reports cancelled instead of partial items.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "stream::cancel_writer",
      "identity": "can.std.stream@1::cancel_writer",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "writer",
          "type": "stream::writer"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "stream::close_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "FileSink.end"
        ],
        "adapter": "Record the reason, then terminal owner close; racing writes complete under their lease.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "files::read_stream",
      "identity": "can.std.files@1::read_stream",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "max_chunk",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "stream::reader<bytes::buffer>",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::limit_exceeded",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.file"
        ],
        "adapter": "Stat at open for acquisition errors, then lazy streaming pulls; later filesystem changes fail reads, not the open.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "files::read_lines_stream",
      "identity": "can.std.files@1::read_lines_stream",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "max_line",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "stream::reader<str>",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::limit_exceeded",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.file",
          "TextDecoder"
        ],
        "adapter": "Stat at open for acquisition errors, then fatal streaming UTF-8 decode with \\n framing, CR tolerance, trailing segment delivery and line caps.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "files::write_stream",
      "identity": "can.std.files@1::write_stream",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "path",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "stream::writer",
      "callbacks": [],
      "emits": [
        "files::not_found",
        "files::denied",
        "files::invalid_path",
        "files::unexpected_kind",
        "files::io_error"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "FileSink"
        ],
        "adapter": "Create or truncate at open with acquisition errors, then lazy accepted-count writes; write-after-end is unreachable through owner close.",
        "task": "B1-05"
      },
      "assertion": "supplied",
      "refs": [
        "B1-05"
      ]
    },
    {
      "name": "url::parse",
      "identity": "can.std.url@1::parse",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "text",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "url::parts",
      "callbacks": [],
      "emits": [
        "url::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Parse absolute http/https URLs into immutable part records; other schemes and malformed text reject, userinfo never projects; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "url::resolve",
      "identity": "can.std.url@1::resolve",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "base",
          "type": "str"
        },
        {
          "name": "input",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "url::parts",
      "callbacks": [],
      "emits": [
        "url::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Resolve relative references against absolute http/https bases per WHATWG URL; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "url::to_string",
      "identity": "can.std.url@1::to_string",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "url::parts"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "url::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL"
        ],
        "adapter": "Serialize part records back to href form; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "url::query_all",
      "identity": "can.std.url@1::query_all",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "url::parts"
        },
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URLSearchParams"
        ],
        "adapter": "Read every form-decoded value for one query key in document order; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "url::query_pairs",
      "identity": "can.std.url@1::query_pairs",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "url::parts"
        }
      ],
      "staticInputs": [],
      "result": "url::query_pair[]",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URLSearchParams"
        ],
        "adapter": "Project every form-decoded query pair in document order, duplicates kept; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "url::with_query",
      "identity": "can.std.url@1::with_query",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "url::parts"
        },
        {
          "name": "pairs",
          "type": "url::query_pair[]"
        }
      ],
      "staticInputs": [],
      "result": "url::parts",
      "callbacks": [],
      "emits": [
        "url::invalid_url"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "URL",
          "URLSearchParams"
        ],
        "adapter": "Rebuild the query string from ordered pairs with form encoding, keeping fragment and parts; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "time::instant_from_epoch_millis",
      "identity": "can.std.time@1::instant_from_epoch_millis",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "millis",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "time::instant",
      "callbacks": [],
      "emits": [
        "time::out_of_range"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Date"
        ],
        "adapter": "Admit epoch milliseconds inside the native Date span as opaque instants; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "time::instant_epoch_millis",
      "identity": "can.std.time@1::instant_epoch_millis",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "instant",
          "type": "time::instant"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "BigInt"
        ],
        "adapter": "Project the exact epoch milliseconds from an instant; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "time::format_in_zone",
      "identity": "can.std.time@1::format_in_zone",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "instant",
          "type": "time::instant"
        },
        {
          "name": "locale",
          "type": "str"
        },
        {
          "name": "zone",
          "type": "str"
        },
        {
          "name": "date_style",
          "type": "str"
        },
        {
          "name": "time_style",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "time::invalid_zone",
        "time::invalid_option"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Intl.DateTimeFormat"
        ],
        "adapter": "Format with explicit locale, IANA zone, and full/long/medium/short/none styles; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "time::resolve_zoned_time",
      "identity": "can.std.time@1::resolve_zoned_time",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "civil",
          "type": "time::civil"
        },
        {
          "name": "zone",
          "type": "str"
        },
        {
          "name": "policy",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "time::instant",
      "callbacks": [],
      "emits": [
        "time::invalid_zone",
        "time::nonexistent_time",
        "time::invalid_option"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Date",
          "Intl.DateTimeFormat"
        ],
        "adapter": "Resolve civil time in a zone with explicit earlier(0)/later(1) DST policy; gaps reject after round-trip verification; execute in ordinary assertions.",
        "task": "B1-13"
      },
      "assertion": "real",
      "refs": [
        "B1-13"
      ]
    },
    {
      "name": "ws::connect",
      "identity": "can.std.ws@1::connect",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "url",
          "type": "str"
        },
        {
          "name": "protocols",
          "type": "str[]"
        },
        {
          "name": "max_message_bytes",
          "type": "int"
        },
        {
          "name": "max_queued_events",
          "type": "int"
        },
        {
          "name": "max_send_bytes",
          "type": "int"
        },
        {
          "name": "deadline_ms",
          "type": "int"
        },
        {
          "name": "insecure_tls",
          "type": "bool"
        }
      ],
      "staticInputs": [],
      "result": "ws::connection",
      "callbacks": [],
      "emits": [
        "ws::connect_failed",
        "ws::invalid_url",
        "ws::invalid_protocol",
        "ws::limit_exceeded"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "WebSocket"
        ],
        "adapter": "Open a client session, wait for the handshake under the caller deadline, and vend the session with its event reader.",
        "task": "B1-07"
      },
      "assertion": "supplied",
      "refs": [
        "B1-07"
      ]
    },
    {
      "name": "ws::accept",
      "identity": "can.std.ws@1::accept",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "request",
          "type": "http::request"
        },
        {
          "name": "protocol",
          "type": "str"
        },
        {
          "name": "max_message_bytes",
          "type": "int"
        },
        {
          "name": "max_queued_events",
          "type": "int"
        },
        {
          "name": "max_send_bytes",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "ws::connection",
      "callbacks": [],
      "emits": [
        "ws::upgrade_failed",
        "ws::unsupported_protocol",
        "ws::invalid_protocol",
        "ws::limit_exceeded"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Bun.serve",
          "Request"
        ],
        "adapter": "Upgrade a routed request after handler authentication and vend the session with its event reader.",
        "task": "B1-07"
      },
      "assertion": "supplied",
      "refs": [
        "B1-07"
      ]
    },
    {
      "name": "ws::send_text",
      "identity": "can.std.ws@1::send_text",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "session",
          "type": "ws::session"
        },
        {
          "name": "text",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "ws::send_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "WebSocket",
          "Bun.serve"
        ],
        "adapter": "Queue one text message; server saturation fails so the caller retries after drain.",
        "task": "B1-07"
      },
      "assertion": "supplied",
      "refs": [
        "B1-07"
      ]
    },
    {
      "name": "ws::send_bytes",
      "identity": "can.std.ws@1::send_bytes",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "session",
          "type": "ws::session"
        },
        {
          "name": "data",
          "type": "bytes::buffer"
        }
      ],
      "staticInputs": [],
      "result": "int",
      "callbacks": [],
      "emits": [
        "ws::send_failed"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "WebSocket",
          "Bun.serve"
        ],
        "adapter": "Queue one binary message; server saturation fails so the caller retries after drain.",
        "task": "B1-07"
      },
      "assertion": "supplied",
      "refs": [
        "B1-07"
      ]
    },
    {
      "name": "ws::close",
      "identity": "can.std.ws@1::close",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "session",
          "type": "ws::session"
        },
        {
          "name": "code",
          "type": "int"
        },
        {
          "name": "reason",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "void",
      "callbacks": [],
      "emits": [
        "ws::invalid_close"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "WebSocket",
          "Bun.serve"
        ],
        "adapter": "Validate the close code and reason on both sides, send the frame, and terminally close the session.",
        "task": "B1-07"
      },
      "assertion": "supplied",
      "refs": [
        "B1-07"
      ]
    },
    {
      "name": "cookie::parse",
      "identity": "can.std.cookie@1::parse",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "header",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "cookie::collection",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "CookieMap"
        ],
        "adapter": "Parse a Cookie header into first-wins lookup over the retained ordered pair list.",
        "task": "B1-09"
      },
      "assertion": "real",
      "refs": [
        "B1-09"
      ]
    },
    {
      "name": "cookie::get",
      "identity": "can.std.cookie@1::get",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "collection",
          "type": "cookie::collection"
        },
        {
          "name": "name",
          "type": "str"
        }
      ],
      "staticInputs": [],
      "result": "option::value<str>",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "CookieMap"
        ],
        "adapter": "Return the first pair value for the name, or none when absent.",
        "task": "B1-09"
      },
      "assertion": "real",
      "refs": [
        "B1-09"
      ]
    },
    {
      "name": "cookie::make",
      "identity": "can.std.cookie@1::make",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "value",
          "type": "str"
        },
        {
          "name": "attributes",
          "type": "cookie::attributes"
        }
      ],
      "staticInputs": [],
      "result": "cookie::cookie",
      "callbacks": [],
      "emits": [
        "cookie::invalid_cookie"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Cookie"
        ],
        "adapter": "Validate the name and expiry, then wrap a native cookie.",
        "task": "B1-09"
      },
      "assertion": "real",
      "refs": [
        "B1-09"
      ]
    },
    {
      "name": "cookie::serialize",
      "identity": "can.std.cookie@1::serialize",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "cookie",
          "type": "cookie::cookie"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "Cookie"
        ],
        "adapter": "Render one Set-Cookie field value from a validated cookie.",
        "task": "B1-09"
      },
      "assertion": "real",
      "refs": [
        "B1-09"
      ]
    },
    {
      "name": "cookie::expire",
      "identity": "can.std.cookie@1::expire",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "name",
          "type": "str"
        },
        {
          "name": "path",
          "type": "str"
        },
        {
          "name": "domain",
          "type": "option::value<str>"
        }
      ],
      "staticInputs": [],
      "result": "cookie::cookie",
      "callbacks": [],
      "emits": [
        "cookie::invalid_cookie"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "CookieMap"
        ],
        "adapter": "Build an epoch-expiry tombstone scoped to the matching path and domain.",
        "task": "B1-09"
      },
      "assertion": "real",
      "refs": [
        "B1-09"
      ]
    },
    {
      "name": "csrf::generate",
      "identity": "can.std.csrf@1::generate",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "secret",
          "type": "str"
        },
        {
          "name": "session_id",
          "type": "str"
        },
        {
          "name": "expires_in_ms",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "str",
      "callbacks": [],
      "emits": [
        "csrf::invalid_config"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "CSRF"
        ],
        "adapter": "Mint a session-bound token with explicit secret and fixed base64url/sha256.",
        "task": "B1-09"
      },
      "assertion": "supplied",
      "refs": [
        "B1-09"
      ]
    },
    {
      "name": "csrf::verify",
      "identity": "can.std.csrf@1::verify",
      "kind": "function",
      "receiver": "",
      "parameters": [],
      "inputs": [
        {
          "name": "secret",
          "type": "str"
        },
        {
          "name": "session_id",
          "type": "str"
        },
        {
          "name": "token",
          "type": "str"
        },
        {
          "name": "max_age_ms",
          "type": "int"
        }
      ],
      "staticInputs": [],
      "result": "bool",
      "callbacks": [],
      "emits": [
        "csrf::invalid_config"
      ],
      "callbackErrors": [],
      "lowering": {
        "native": [
          "CSRF"
        ],
        "adapter": "Verify a token against explicit secret, session and age; token faults answer false.",
        "task": "B1-09"
      },
      "assertion": "real",
      "refs": [
        "B1-09"
      ]
    }
  ],
  "nativeDeclarations": [
    {
      "name": "noul",
      "emits": [
        "ai::invalid_question",
        "ai::invalid_answer"
      ],
      "conditionalEmits": [],
      "unionBounds": [
        "handlers"
      ],
      "task": "I27",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "real"
    },
    {
      "name": "choice",
      "emits": [
        "ai::invalid_question",
        "ai::invalid_answer"
      ],
      "conditionalEmits": [],
      "unionBounds": [
        "handlers"
      ],
      "task": "I27",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "real"
    },
    {
      "name": "record_choice",
      "emits": [
        "ai::invalid_question",
        "ai::invalid_answer"
      ],
      "conditionalEmits": [],
      "unionBounds": [
        "handlers"
      ],
      "task": "I27",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "real"
    },
    {
      "name": "score",
      "emits": [
        "ai::invalid_question",
        "ai::invalid_answer"
      ],
      "conditionalEmits": [],
      "unionBounds": [
        "handlers"
      ],
      "task": "I27",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "real"
    },
    {
      "name": "record_score",
      "emits": [
        "ai::invalid_question",
        "ai::invalid_answer"
      ],
      "conditionalEmits": [],
      "unionBounds": [
        "handlers"
      ],
      "task": "I27",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "real"
    },
    {
      "name": "choice_arm",
      "emits": [],
      "conditionalEmits": [],
      "unionBounds": [
        "body"
      ],
      "task": "I27",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "real"
    },
    {
      "name": "judge",
      "emits": [
        "http::request_failed",
        "ai::invalid_question",
        "ai::invalid_answer"
      ],
      "conditionalEmits": [],
      "unionBounds": [
        "questions",
        "handlers",
        "continuation"
      ],
      "task": "LF10",
      "refs": [
        "A2.4"
      ],
      "assertion": "raw-provider"
    },
    {
      "name": "fetch_body",
      "emits": [
        "http::request_failed"
      ],
      "conditionalEmits": [],
      "unionBounds": [],
      "task": "LF10",
      "refs": [
        "A2.4"
      ],
      "assertion": "raw-provider"
    },
    {
      "name": "fetch_envelope",
      "emits": [
        "http::request_failed"
      ],
      "conditionalEmits": [],
      "unionBounds": [],
      "task": "LF10",
      "refs": [
        "A2.4"
      ],
      "assertion": "raw-provider"
    },
    {
      "name": "llm",
      "emits": [
        "http::invalid_request",
        "http::transport_failed",
        "http::timeout",
        "http::body_limit",
        "http::status_error",
        "codec::invalid_data",
        "llm::refused",
        "llm::truncated",
        "llm::invalid_response"
      ],
      "conditionalEmits": [
        {
          "condition": "authenticated",
          "emits": [
            "http::credentials_missing"
          ]
        }
      ],
      "unionBounds": [],
      "task": "I28",
      "refs": [
        "A2",
        "A3"
      ],
      "assertion": "raw-provider"
    }
  ]
} as const);
export const catalogueTypeShapes = freeze([
  {
    "name": "choice_option",
    "identity": "can.prelude@1::choice_option",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "key",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "description",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "standard_failure",
    "identity": "can.prelude@1::standard_failure",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "option::none",
    "identity": "can.std.option@1::none",
    "kind": "record",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "option::some",
    "identity": "can.std.option@1::some",
    "kind": "record",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [
      {
        "name": "value",
        "type": {
          "name": "T",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "option::value",
    "identity": "can.std.option@1::value",
    "kind": "variant",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [],
    "leaves": [
      {
        "name": "option::none",
        "arguments": null
      },
      {
        "name": "option::some",
        "arguments": [
          {
            "name": "T",
            "arguments": null
          }
        ]
      }
    ]
  },
  {
    "name": "number::division",
    "identity": "can.std.number@1::division",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "quotient",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "remainder",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "number::rounded",
    "identity": "can.std.number@1::rounded",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "value",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "remainder_numerator",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "denominator",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "collections::entry",
    "identity": "can.std.collections@1::entry",
    "kind": "record",
    "parameters": [
      {
        "name": "K",
        "constraint": "map_key"
      },
      {
        "name": "V",
        "constraint": "data"
      }
    ],
    "fields": [
      {
        "name": "key",
        "type": {
          "name": "K",
          "arguments": null
        }
      },
      {
        "name": "value",
        "type": {
          "name": "V",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "collections::map",
    "identity": "can.std.collections@1::map",
    "kind": "opaque",
    "parameters": [
      {
        "name": "K",
        "constraint": "map_key"
      },
      {
        "name": "V",
        "constraint": "data"
      }
    ],
    "fields": [],
    "leaves": []
  },
  {
    "name": "collections::set",
    "identity": "can.std.collections@1::set",
    "kind": "opaque",
    "parameters": [
      {
        "name": "K",
        "constraint": "map_key"
      }
    ],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::header",
    "identity": "can.std.http@1::header",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "name",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "value",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::sse_event",
    "identity": "can.std.http@1::sse_event",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "data",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "event",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "id",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "retry",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::multipart_form",
    "identity": "can.std.http@1::multipart_form",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "fields",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "http::multipart_field",
              "arguments": null
            }
          ]
        }
      },
      {
        "name": "files",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "http::multipart_file",
              "arguments": null
            }
          ]
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::multipart_field",
    "identity": "can.std.http@1::multipart_field",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "name",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "value",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::multipart_file",
    "identity": "can.std.http@1::multipart_file",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "name",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "filename",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "content_type",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "content",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::response",
    "identity": "can.std.http@1::response",
    "kind": "record",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [
      {
        "name": "status",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "headers",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "http::header",
              "arguments": null
            }
          ]
        }
      },
      {
        "name": "body",
        "type": {
          "name": "T",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::failure_detail",
    "identity": "can.std.http@1::failure_detail",
    "kind": "variant",
    "parameters": [],
    "fields": [],
    "leaves": [
      {
        "name": "http::invalid_request",
        "arguments": null
      },
      {
        "name": "http::credentials_missing",
        "arguments": null
      },
      {
        "name": "http::transport_failed",
        "arguments": null
      },
      {
        "name": "http::timeout",
        "arguments": null
      },
      {
        "name": "http::body_limit",
        "arguments": null
      },
      {
        "name": "http::status_error",
        "arguments": null
      },
      {
        "name": "codec::invalid_data",
        "arguments": null
      }
    ]
  },
  {
    "name": "bytes::buffer",
    "identity": "can.std.bytes@1::buffer",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "html::node",
    "identity": "can.std.html@1::node",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "html::safe",
    "identity": "can.std.html@1::safe",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "html::url",
    "identity": "can.std.html@1::url",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "html::attribute",
    "identity": "can.std.html@1::attribute",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "html::tag",
    "identity": "can.std.html@1::tag",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "htmx::target",
    "identity": "can.std.htmx@1::target",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "htmx::swap",
    "identity": "can.std.htmx@1::swap",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::request",
    "identity": "can.std.http@1::request",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::server_response",
    "identity": "can.std.http@1::server_response",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::status",
    "identity": "can.std.http@1::status",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::body_status",
    "identity": "can.std.http@1::body_status",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::server_headers",
    "identity": "can.std.http@1::server_headers",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::route",
    "identity": "can.std.http@1::route",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::router",
    "identity": "can.std.http@1::router",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::server",
    "identity": "can.std.http@1::server",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::server_config",
    "identity": "can.std.http@1::server_config",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "http::tls_config",
    "identity": "can.std.http@1::tls_config",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "sql::pool",
    "identity": "can.std.sql@1::pool",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "sql::transaction",
    "identity": "can.std.sql@1::transaction",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "sql::sqlite_file_options",
    "identity": "can.std.sql@1::sqlite_file_options",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "mode",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "busy_timeout_ms",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::commit",
    "identity": "can.std.sql@1::commit",
    "kind": "record",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [
      {
        "name": "value",
        "type": {
          "name": "T",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::rollback",
    "identity": "can.std.sql@1::rollback",
    "kind": "record",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [
      {
        "name": "value",
        "type": {
          "name": "T",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::decision",
    "identity": "can.std.sql@1::decision",
    "kind": "variant",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [],
    "leaves": [
      {
        "name": "sql::commit",
        "arguments": [
          {
            "name": "T",
            "arguments": null
          }
        ]
      },
      {
        "name": "sql::rollback",
        "arguments": [
          {
            "name": "T",
            "arguments": null
          }
        ]
      }
    ]
  },
  {
    "name": "files::file_info",
    "identity": "can.std.files@1::file_info",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "kind",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "size",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::entry",
    "identity": "can.std.files@1::entry",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "kind",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::options",
    "identity": "can.std.process@1::options",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "cwd",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "inherit_env",
        "type": {
          "name": "bool",
          "arguments": null
        }
      },
      {
        "name": "env",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "str",
              "arguments": null
            }
          ]
        }
      },
      {
        "name": "stdin",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      },
      {
        "name": "stdout_limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "stderr_limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "deadline_ms",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "grace_ms",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::result",
    "identity": "can.std.process@1::result",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "stdout",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      },
      {
        "name": "stderr",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      },
      {
        "name": "code",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "signal",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "stream::reader",
    "identity": "can.std.stream@1::reader",
    "kind": "opaque",
    "parameters": [
      {
        "name": "T",
        "constraint": "data"
      }
    ],
    "fields": [],
    "leaves": []
  },
  {
    "name": "stream::writer",
    "identity": "can.std.stream@1::writer",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "crypto::key",
    "identity": "can.std.crypto@1::key",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "crypto::keypair",
    "identity": "can.std.crypto@1::keypair",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "private_key",
        "type": {
          "name": "crypto::key",
          "arguments": null
        }
      },
      {
        "name": "public_key",
        "type": {
          "name": "crypto::key",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "crypto::sealed",
    "identity": "can.std.crypto@1::sealed",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "nonce",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      },
      {
        "name": "ciphertext",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "url::parts",
    "identity": "can.std.url@1::parts",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "scheme",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "host",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "port",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "query",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "fragment",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "url::query_pair",
    "identity": "can.std.url@1::query_pair",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "name",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "value",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "text::regex",
    "identity": "can.std.text@1::regex",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "text::regex_match",
    "identity": "can.std.text@1::regex_match",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "text",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "start",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "end",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "groups",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "str",
              "arguments": null
            }
          ]
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "time::instant",
    "identity": "can.std.time@1::instant",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "time::civil",
    "identity": "can.std.time@1::civil",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "year",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "month",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "day",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "hour",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "minute",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "second",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "millisecond",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::session",
    "identity": "can.std.ws@1::session",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "ws::connection",
    "identity": "can.std.ws@1::connection",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "session",
        "type": {
          "name": "ws::session",
          "arguments": null
        }
      },
      {
        "name": "events",
        "type": {
          "name": "stream::reader",
          "arguments": [
            {
              "name": "ws::event",
              "arguments": null
            }
          ]
        }
      },
      {
        "name": "protocol",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::event",
    "identity": "can.std.ws@1::event",
    "kind": "variant",
    "parameters": [],
    "fields": [],
    "leaves": [
      {
        "name": "ws::text",
        "arguments": null
      },
      {
        "name": "ws::binary",
        "arguments": null
      },
      {
        "name": "ws::drain",
        "arguments": null
      },
      {
        "name": "ws::closed",
        "arguments": null
      }
    ]
  },
  {
    "name": "ws::text",
    "identity": "can.std.ws@1::text",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "text",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::binary",
    "identity": "can.std.ws@1::binary",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "data",
        "type": {
          "name": "bytes::buffer",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::drain",
    "identity": "can.std.ws@1::drain",
    "kind": "record",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "ws::closed",
    "identity": "can.std.ws@1::closed",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "code",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "cookie::attributes",
    "identity": "can.std.cookie@1::attributes",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "domain",
        "type": {
          "name": "option::value",
          "arguments": [
            {
              "name": "str",
              "arguments": null
            }
          ]
        }
      },
      {
        "name": "secure",
        "type": {
          "name": "bool",
          "arguments": null
        }
      },
      {
        "name": "http_only",
        "type": {
          "name": "bool",
          "arguments": null
        }
      },
      {
        "name": "same_site",
        "type": {
          "name": "cookie::same_site",
          "arguments": null
        }
      },
      {
        "name": "max_age",
        "type": {
          "name": "option::value",
          "arguments": [
            {
              "name": "int",
              "arguments": null
            }
          ]
        }
      },
      {
        "name": "expires_ms",
        "type": {
          "name": "option::value",
          "arguments": [
            {
              "name": "int",
              "arguments": null
            }
          ]
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "cookie::same_site",
    "identity": "can.std.cookie@1::same_site",
    "kind": "variant",
    "parameters": [],
    "fields": [],
    "leaves": [
      {
        "name": "cookie::strict",
        "arguments": null
      },
      {
        "name": "cookie::lax",
        "arguments": null
      },
      {
        "name": "cookie::none",
        "arguments": null
      }
    ]
  },
  {
    "name": "cookie::strict",
    "identity": "can.std.cookie@1::strict",
    "kind": "record",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "cookie::lax",
    "identity": "can.std.cookie@1::lax",
    "kind": "record",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "cookie::none",
    "identity": "can.std.cookie@1::none",
    "kind": "record",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "cookie::cookie",
    "identity": "can.std.cookie@1::cookie",
    "kind": "opaque",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "cookie::collection",
    "identity": "can.std.cookie@1::collection",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "pairs",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "cookie::pair",
              "arguments": null
            }
          ]
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "cookie::pair",
    "identity": "can.std.cookie@1::pair",
    "kind": "record",
    "parameters": [],
    "fields": [
      {
        "name": "name",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "value",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "all_failed",
    "identity": "can.prelude@1::all_failed",
    "kind": "error",
    "parameters": [
      {
        "name": "F",
        "constraint": "failure_variant"
      }
    ],
    "fields": [
      {
        "name": "failures",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "F",
              "arguments": null
            }
          ]
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "number::inexact",
    "identity": "can.std.number@1::inexact",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "text::invalid_number",
    "identity": "can.std.text@1::invalid_number",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "input",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "text::invalid_bool",
    "identity": "can.std.text@1::invalid_bool",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "input",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "number::invalid_bool",
    "identity": "can.std.number@1::invalid_bool",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "value",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "text::empty_separator",
    "identity": "can.std.text@1::empty_separator",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "text::empty_pattern",
    "identity": "can.std.text@1::empty_pattern",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "text::invalid_unicode",
    "identity": "can.std.text@1::invalid_unicode",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "collections::key_absent",
    "identity": "can.std.collections@1::key_absent",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "collections::key_exists",
    "identity": "can.std.collections@1::key_exists",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "number::zero_divisor",
    "identity": "can.std.number@1::zero_divisor",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "checks::failed",
    "identity": "can.std.checks@1::failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::invalid_request",
    "identity": "can.std.http@1::invalid_request",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::credentials_missing",
    "identity": "can.std.http@1::credentials_missing",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "variable",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::transport_failed",
    "identity": "can.std.http@1::transport_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "phase",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::timeout",
    "identity": "can.std.http@1::timeout",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "timeout_ms",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::body_limit",
    "identity": "can.std.http@1::body_limit",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::status_error",
    "identity": "can.std.http@1::status_error",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "status",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "headers",
        "type": {
          "name": "[]",
          "arguments": [
            {
              "name": "http::header",
              "arguments": null
            }
          ]
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::request_failed",
    "identity": "can.std.http@1::request_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "detail",
        "type": {
          "name": "http::failure_detail",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "codec::invalid_data",
    "identity": "can.std.codec@1::invalid_data",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ai::invalid_question",
    "identity": "can.std.ai@1::invalid_question",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ai::invalid_answer",
    "identity": "can.std.ai@1::invalid_answer",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "question",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "llm::refused",
    "identity": "can.std.llm@1::refused",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "llm::truncated",
    "identity": "can.std.llm@1::truncated",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "llm::invalid_response",
    "identity": "can.std.llm@1::invalid_response",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "io::read_failed",
    "identity": "can.std.io@1::read_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "io::write_failed",
    "identity": "can.std.io@1::write_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "io::limit_exceeded",
    "identity": "can.std.io@1::limit_exceeded",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "html::invalid_structure",
    "identity": "can.std.html@1::invalid_structure",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "html::invalid_url",
    "identity": "can.std.html@1::invalid_url",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "htmx::invalid_target",
    "identity": "can.std.htmx@1::invalid_target",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "htmx::invalid_interval",
    "identity": "can.std.htmx@1::invalid_interval",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "milliseconds",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::invalid_route",
    "identity": "can.std.http@1::invalid_route",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::duplicate_route",
    "identity": "can.std.http@1::duplicate_route",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "method",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::ambiguous_route",
    "identity": "can.std.http@1::ambiguous_route",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "first",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "second",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::invalid_server_config",
    "identity": "can.std.http@1::invalid_server_config",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::bind_failed",
    "identity": "can.std.http@1::bind_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "address",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "http::shutdown_failed",
    "identity": "can.std.http@1::shutdown_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "phase",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::connection_failed",
    "identity": "can.std.sql@1::connection_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "phase",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::query_failed",
    "identity": "can.std.sql@1::query_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "code",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::row_missing",
    "identity": "can.std.sql@1::row_missing",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "query",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::row_count",
    "identity": "can.std.sql@1::row_count",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "query",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "actual",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::schema_mismatch",
    "identity": "can.std.sql@1::schema_mismatch",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::constraint_failed",
    "identity": "can.std.sql@1::constraint_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "constraint",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::transaction_failed",
    "identity": "can.std.sql@1::transaction_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "phase",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::commit_unknown",
    "identity": "can.std.sql@1::commit_unknown",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "transaction_id",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::close_failed",
    "identity": "can.std.sql@1::close_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::row_limit",
    "identity": "can.std.sql@1::row_limit",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "sql::unsupported_value",
    "identity": "can.std.sql@1::unsupported_value",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "clock::invalid_duration",
    "identity": "can.std.clock@1::invalid_duration",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "milliseconds",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "random::invalid_length",
    "identity": "can.std.random@1::invalid_length",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "length",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "env::invalid_name",
    "identity": "can.std.env@1::invalid_name",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "name",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "log::write_failed",
    "identity": "can.std.log@1::write_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "level",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::not_found",
    "identity": "can.std.files@1::not_found",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::denied",
    "identity": "can.std.files@1::denied",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::already_exists",
    "identity": "can.std.files@1::already_exists",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::invalid_path",
    "identity": "can.std.files@1::invalid_path",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::io_error",
    "identity": "can.std.files@1::io_error",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::limit_exceeded",
    "identity": "can.std.files@1::limit_exceeded",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::not_empty",
    "identity": "can.std.files@1::not_empty",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::cross_device",
    "identity": "can.std.files@1::cross_device",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "source",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "destination",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "files::unexpected_kind",
    "identity": "can.std.files@1::unexpected_kind",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "path",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::spawn_failed",
    "identity": "can.std.process@1::spawn_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "executable",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::timeout",
    "identity": "can.std.process@1::timeout",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "deadline_ms",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::output_limit",
    "identity": "can.std.process@1::output_limit",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "stream",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::nonzero",
    "identity": "can.std.process@1::nonzero",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "code",
        "type": {
          "name": "int",
          "arguments": null
        }
      },
      {
        "name": "signal",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::invalid_config",
    "identity": "can.std.process@1::invalid_config",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "field",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "process::io_error",
    "identity": "can.std.process@1::io_error",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "stream::read_failed",
    "identity": "can.std.stream@1::read_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "stream::write_failed",
    "identity": "can.std.stream@1::write_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "stream::cancelled",
    "identity": "can.std.stream@1::cancelled",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "stream::close_failed",
    "identity": "can.std.stream@1::close_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "password::cost_rejected",
    "identity": "can.std.password@1::cost_rejected",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "profile",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "password::invalid_hash",
    "identity": "can.std.password@1::invalid_hash",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "crypto::invalid_key",
    "identity": "can.std.crypto@1::invalid_key",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "crypto::invalid_nonce",
    "identity": "can.std.crypto@1::invalid_nonce",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "length",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "crypto::key_misuse",
    "identity": "can.std.crypto@1::key_misuse",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "operation",
        "type": {
          "name": "str",
          "arguments": null
        }
      },
      {
        "name": "algorithm",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "crypto::decrypt_failed",
    "identity": "can.std.crypto@1::decrypt_failed",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "url::invalid_url",
    "identity": "can.std.url@1::invalid_url",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "text::invalid_regex",
    "identity": "can.std.text@1::invalid_regex",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "text::invalid_limit",
    "identity": "can.std.text@1::invalid_limit",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "time::out_of_range",
    "identity": "can.std.time@1::out_of_range",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "millis",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "time::invalid_zone",
    "identity": "can.std.time@1::invalid_zone",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "zone",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "time::nonexistent_time",
    "identity": "can.std.time@1::nonexistent_time",
    "kind": "error",
    "parameters": [],
    "fields": [],
    "leaves": []
  },
  {
    "name": "time::invalid_option",
    "identity": "can.std.time@1::invalid_option",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::connect_failed",
    "identity": "can.std.ws@1::connect_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::upgrade_failed",
    "identity": "can.std.ws@1::upgrade_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::unsupported_protocol",
    "identity": "can.std.ws@1::unsupported_protocol",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "protocol",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::send_failed",
    "identity": "can.std.ws@1::send_failed",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::invalid_close",
    "identity": "can.std.ws@1::invalid_close",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::limit_exceeded",
    "identity": "can.std.ws@1::limit_exceeded",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "limit",
        "type": {
          "name": "int",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::invalid_url",
    "identity": "can.std.ws@1::invalid_url",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "ws::invalid_protocol",
    "identity": "can.std.ws@1::invalid_protocol",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "protocol",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "cookie::invalid_cookie",
    "identity": "can.std.cookie@1::invalid_cookie",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  },
  {
    "name": "csrf::invalid_config",
    "identity": "can.std.csrf@1::invalid_config",
    "kind": "error",
    "parameters": [],
    "fields": [
      {
        "name": "reason",
        "type": {
          "name": "str",
          "arguments": null
        }
      }
    ],
    "leaves": []
  }
] as const);
export type CatalogueOperationName = typeof catalogue.operations[number]["name"];
export type CatalogueTypeName = typeof catalogue.types[number]["name"];
export type CatalogueErrorName = typeof catalogue.errors[number]["name"];
export type CatalogueErrorIdentity = {
  readonly name: CatalogueErrorName;
  readonly identity: string;
  readonly id: number;
  readonly typeArguments: readonly string[];
};
export function operation(name: string, target: string = catalogue.targetId, revision: number = catalogue.revision) {
  if (target !== catalogue.targetId || revision !== catalogue.revision) throw new Error("unsupported catalogue target or revision");
  const found = catalogue.operations.find(value => value.name === name);
  if (!found) throw new Error("unknown catalogue operation; host registration is closed");
  return found;
}
export function requireConstructor(name: string) {
  const type = catalogue.types.find(value => value.name === name);
  if (type) {
    if (!type.constructible) throw new Error("catalogue type has no public constructor");
    return type;
  }
  const error = catalogue.errors.find(value => value.name === name);
  if (!error) throw new Error("unknown catalogue constructor");
  return error;
}
export function errorIdentity(name: string, typeArguments: readonly string[] = []): CatalogueErrorIdentity {
  const error = catalogue.errors.find(value => value.name === name);
  if (!error) throw new Error("unallocated catalogue error");
  const argumentsCopy = dataArray(typeArguments);
  if (error.parameters.length !== argumentsCopy.length || argumentsCopy.some(value => typeof value !== "string" || value.length === 0)) throw new Error("invalid error specialization");
  return freeze({ name: error.name, identity: error.identity, id: error.id, typeArguments: argumentsCopy as string[] });
}
export function validateErrorIdentity(value: CatalogueErrorIdentity): CatalogueErrorIdentity {
  const keys = dataKeys(value);
  if (keys.length !== 4 || keys.some(key => typeof key !== "string" || !["name", "identity", "id", "typeArguments"].includes(key))) throw new Error("invalid catalogue identity shape");
  const expected = errorIdentity(dataProperty(value, "name") as string, dataProperty(value, "typeArguments") as string[]);
  if (dataProperty(value, "identity") !== expected.identity || dataProperty(value, "id") !== expected.id) throw new Error("catalogue error identity mismatch");
  return expected;
}
