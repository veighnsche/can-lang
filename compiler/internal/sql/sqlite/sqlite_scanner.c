/*
 * External scanner for tree-sitter-sqlite3.
 *
 * Tree-sitter's regex-based lexer is parser-state-driven and cannot
 * reject malformed tokens that have a valid alternative interpretation.
 * For example, `X'01001'` (odd-length hex blob) falls back to
 * (X-as-identifier, '01001'-as-string) because the strict blob_literal
 * regex doesn't match it. sqlite's tokenizer, by contrast, rejects
 * `X'01001'` outright at tokenize-time.
 *
 * This scanner closes that gap by detecting MALFORMED forms and
 * emitting poison tokens that no parser rule consumes — which forces
 * an ERROR node at the parse position. The external symbols are:
 *
 *   MALFORMED_BLOB         X'<bad>'    (odd length, non-hex char,
 *                                       embedded space, etc.)
 *   MALFORMED_NUMBER_ID    123abc      (numeric immediately followed
 *                                       by identifier — sqlite emits
 *                                       "unrecognized token: 123abc")
 *
 * The scanner is conservative: it only emits these tokens when the
 * input is unambiguously malformed. Valid forms are left for the
 * regex-based tokens (blob_literal / numeric_literal) to handle.
 */

#include "tree_sitter/parser.h"

#include <ctype.h>
#include <stdbool.h>

/* Order MUST match the `externals` array in grammar.js. */
enum TokenType {
    MALFORMED_BLOB_LITERAL,
    MALFORMED_NUMBER_ID,
};

static inline bool is_hex(int32_t c) {
    return (c >= '0' && c <= '9') ||
           (c >= 'a' && c <= 'f') ||
           (c >= 'A' && c <= 'F');
}

static inline bool is_id_continue(int32_t c) {
    return (c >= 'A' && c <= 'Z') ||
           (c >= 'a' && c <= 'z') ||
           (c >= '0' && c <= '9') ||
           c == '_' || c == '$' ||
           // sqlite tokenize.c IdChar: any byte >= 0x80
           c >= 0x80;
}

/* X'<hexpairs>' — emit MALFORMED_BLOB iff the form looks blob-shaped
 * (X' or x' followed by chars and a closing ') AND the content is not
 * a valid even-length hex string. */
static bool scan_malformed_blob(TSLexer *lexer) {
    int32_t c = lexer->lookahead;
    if (c != 'X' && c != 'x') return false;
    lexer->advance(lexer, false);
    if (lexer->lookahead != '\'') return false;
    lexer->advance(lexer, false);

    /* Walk to the closing apostrophe, tracking whether content is a
     * pure even-length hex string. */
    int n = 0;
    bool ok = true;
    while (lexer->lookahead != '\'' && lexer->lookahead != 0) {
        if (!is_hex(lexer->lookahead)) ok = false;
        lexer->advance(lexer, false);
        n++;
    }
    if (lexer->lookahead != '\'') return false;  /* unterminated */
    lexer->advance(lexer, false);                /* consume closing ' */

    if (ok && (n % 2) == 0) {
        /* Valid blob — let the regex-based blob_literal token handle
         * it. We've already advanced past the close quote, but since
         * we're not calling mark_end() the lexer position rewinds on
         * a false return. */
        return false;
    }

    lexer->mark_end(lexer);
    lexer->result_symbol = MALFORMED_BLOB_LITERAL;
    return true;
}

/* Numeric literal directly followed by an identifier-continuing char.
 * sqlite emits "unrecognized token: 123abc" for this. */
static bool scan_malformed_number_id(TSLexer *lexer) {
    int32_t c = lexer->lookahead;
    bool started_with_digit = (c >= '0' && c <= '9');
    bool started_with_dot = (c == '.');
    if (!started_with_digit && !started_with_dot) return false;

    if (started_with_dot) {
        lexer->advance(lexer, false);
        if (lexer->lookahead < '0' || lexer->lookahead > '9') return false;
    }

    /* Consume digits / hex / dot / e / sign / underscore — a permissive
     * numeric scan; we just need to find where the number "ends" so
     * we can check the next char. */
    bool saw_dot = started_with_dot;
    bool saw_e = false;
    bool is_hex_lit = false;

    /* Track whether the numeric run ends with `_` or has consecutive
     * `__` — both are sqlite-rejected forms.  We accept underscore in
     * the run-loops below to capture the malformed-shape, then check
     * for these violations after. */
    bool trailing_underscore = false;
    bool consecutive_underscore = false;
    int32_t prev = 0;
    #define ADV_NUM_DIGIT(p) do { \
        prev = (p); \
        lexer->advance(lexer, false); \
    } while (0)

    if (lexer->lookahead == '0') {
        ADV_NUM_DIGIT('0');
        if (lexer->lookahead == 'x' || lexer->lookahead == 'X') {
            is_hex_lit = true;
            ADV_NUM_DIGIT(lexer->lookahead);
            while (is_hex(lexer->lookahead) || lexer->lookahead == '_') {
                if (lexer->lookahead == '_' && prev == '_') consecutive_underscore = true;
                ADV_NUM_DIGIT(lexer->lookahead);
            }
            trailing_underscore = (prev == '_');
        }
    }
    if (!is_hex_lit) {
        while ((lexer->lookahead >= '0' && lexer->lookahead <= '9') ||
               lexer->lookahead == '_') {
            if (lexer->lookahead == '_' && prev == '_') consecutive_underscore = true;
            ADV_NUM_DIGIT(lexer->lookahead);
        }
        if (lexer->lookahead == '.' && !saw_dot && !saw_e) {
            saw_dot = true;
            ADV_NUM_DIGIT('.');
            while ((lexer->lookahead >= '0' && lexer->lookahead <= '9') ||
                   lexer->lookahead == '_') {
                if (lexer->lookahead == '_' && prev == '_') consecutive_underscore = true;
                ADV_NUM_DIGIT(lexer->lookahead);
            }
        }
        if ((lexer->lookahead == 'e' || lexer->lookahead == 'E') && !saw_e) {
            saw_e = true;
            ADV_NUM_DIGIT(lexer->lookahead);
            bool exp_signed = false;
            if (lexer->lookahead == '+' || lexer->lookahead == '-') {
                exp_signed = true;
                ADV_NUM_DIGIT(lexer->lookahead);
            }
            int exp_digits = 0;
            while ((lexer->lookahead >= '0' && lexer->lookahead <= '9') ||
                   lexer->lookahead == '_') {
                if (lexer->lookahead == '_' && prev == '_') consecutive_underscore = true;
                if (lexer->lookahead != '_') exp_digits++;
                ADV_NUM_DIGIT(lexer->lookahead);
            }
            /* Scientific suffix `e[+-]?` MUST be followed by at least
             * one digit — sqlite rejects `1.0e`, `1.0e,`, `1.0e+`. */
            if (exp_digits == 0) {
                lexer->mark_end(lexer);
                lexer->result_symbol = MALFORMED_NUMBER_ID;
                return true;
            }
            (void)exp_signed;
        }
        trailing_underscore = (prev == '_');
    }
    #undef ADV_NUM_DIGIT

    /* Number directly fused to identifier (`123abc`), or numeric
     * literal with malformed underscore placement (`0xFFEF_`,
     * `123__456`) — sqlite emits "unrecognized token" for both. */
    bool fused_id = is_id_continue(lexer->lookahead);
    if (fused_id) {
        while (is_id_continue(lexer->lookahead)) {
            lexer->advance(lexer, false);
        }
    }
    if (fused_id || trailing_underscore || consecutive_underscore) {
        lexer->mark_end(lexer);
        lexer->result_symbol = MALFORMED_NUMBER_ID;
        return true;
    }

    /* Clean number — let the regex-based numeric_literal token
     * handle it. We rewind by not calling mark_end. */
    return false;
}

/* --- tree-sitter ABI ---------------------------------------------- */

void *tree_sitter_sqlite3_external_scanner_create(void) {
    return NULL;
}

void tree_sitter_sqlite3_external_scanner_destroy(void *payload) {
    (void)payload;
}

unsigned tree_sitter_sqlite3_external_scanner_serialize(void *payload, char *buffer) {
    (void)payload;
    (void)buffer;
    return 0;
}

void tree_sitter_sqlite3_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {
    (void)payload;
    (void)buffer;
    (void)length;
}

bool tree_sitter_sqlite3_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
    (void)payload;

    /* Skip leading whitespace — tree-sitter normally handles extras
     * before invoking the scanner, but be defensive. */
    while (lexer->lookahead == ' ' || lexer->lookahead == '\t' ||
           lexer->lookahead == '\n' || lexer->lookahead == '\r') {
        lexer->advance(lexer, true);
    }

    /* At most one scan path per call. Tree-sitter rewinds the lexer
     * state when scan() returns false (advance() calls are tentative
     * unless committed by mark_end()). But if we tried multiple
     * scanners in one call, a failed first scanner would leak its
     * advance() into the second scanner's view of the input. So
     * dispatch on lookahead first. */
    int32_t c = lexer->lookahead;
    if (valid_symbols[MALFORMED_BLOB_LITERAL] && (c == 'X' || c == 'x')) {
        return scan_malformed_blob(lexer);
    }
    if (valid_symbols[MALFORMED_NUMBER_ID] &&
        ((c >= '0' && c <= '9') || c == '.')) {
        return scan_malformed_number_id(lexer);
    }
    return false;
}
