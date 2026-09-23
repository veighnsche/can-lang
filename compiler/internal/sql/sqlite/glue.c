// Iterative concrete-syntax-tree dump for the vendored SQLite grammar.
// The walker emits one line per node in pre-order:
//
//	<depth>\x1fnamed\x1f<type>\x1f<start>\x1f<end>\n
//
// named is 1 for named nodes and 0 for anonymous tokens. Depths rebuild
// the tree on the Go side; spans slice the original source there, so no
// source text crosses this boundary. Node type bytes outside printable
// ASCII are replaced with '?' so the framing cannot break: every type the
// adapter matches is plain ASCII. The walk is iterative over a heap stack
// and stops after 1M nodes with truncated set rather than growing
// without bound on hostile input.
#include <tree_sitter/api.h>
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

const TSLanguage *tree_sitter_sqlite3(void);

#define SQLITE_DUMP_NODE_CAP 1000000

typedef struct {
	TSNode node;
	uint32_t next;
	uint32_t depth;
} dump_frame;

char *sqlite_dump(const char *src, unsigned srclen, int *has_error, int *truncated) {
	TSParser *parser = ts_parser_new();
	if (!ts_parser_set_language(parser, tree_sitter_sqlite3())) {
		ts_parser_delete(parser);
		return NULL;
	}
	TSTree *tree = ts_parser_parse_string(parser, NULL, src, srclen);
	*has_error = ts_node_has_error(ts_tree_root_node(tree)) ? 1 : 0;
	*truncated = 0;

	size_t cap = 65536, len = 0;
	char *buf = malloc(cap);
	if (!buf) {
		ts_tree_delete(tree);
		ts_parser_delete(parser);
		return NULL;
	}
	size_t stack_cap = 256, stack_len = 0;
	dump_frame *stack = malloc(stack_cap * sizeof *stack);
	if (!stack) {
		free(buf);
		ts_tree_delete(tree);
		ts_parser_delete(parser);
		return NULL;
	}
	stack[stack_len++] = (dump_frame){ts_tree_root_node(tree), 0, 0};
	uint32_t emitted = 0;
	char line[512];
	while (stack_len > 0) {
		dump_frame *top = &stack[stack_len - 1];
		uint32_t kids = ts_node_child_count(top->node);
		if (top->next == 0) {
			if (emitted >= SQLITE_DUMP_NODE_CAP) {
				*truncated = 1;
				break;
			}
			const char *type = ts_node_type(top->node);
			char clean[128];
			size_t ci = 0;
			for (const char *p = type; *p && ci + 1 < sizeof clean; p++) {
				unsigned char c = (unsigned char)*p;
				clean[ci++] = (c >= 0x20 && c < 0x7f) ? (char)c : '?';
			}
			clean[ci] = 0;
			int n = snprintf(line, sizeof line, "%u\x1f%d\x1f%s\x1f%u\x1f%u\n",
				top->depth, ts_node_is_named(top->node) ? 1 : 0, clean,
				ts_node_start_byte(top->node), ts_node_end_byte(top->node));
			if (n < 0 || (size_t)n >= sizeof line) {
				*truncated = 1;
				break;
			}
			if (len + (size_t)n + 1 > cap) {
				cap = (len + (size_t)n + 1) * 2;
				char *grown = realloc(buf, cap);
				if (!grown) {
					free(buf);
					buf = NULL;
					break;
				}
				buf = grown;
				top = &stack[stack_len - 1];
			}
			memcpy(buf + len, line, (size_t)n);
			len += (size_t)n;
			emitted++;
		}
		if (top->next < kids) {
			if (stack_len >= stack_cap) {
				stack_cap *= 2;
				dump_frame *grown = realloc(stack, stack_cap * sizeof *grown);
				if (!grown) {
					*truncated = 1;
					break;
				}
				stack = grown;
				top = &stack[stack_len - 1];
			}
			TSNode kid = ts_node_child(top->node, top->next);
			top->next++;
			stack[stack_len++] = (dump_frame){kid, 0, top->depth + 1};
		} else {
			stack_len--;
		}
	}
	if (buf) {
		if (len + 1 > cap) {
			char *grown = realloc(buf, len + 1);
			if (!grown) {
				free(buf);
				buf = NULL;
			} else {
				buf = grown;
			}
		}
		if (buf) {
			buf[len] = 0;
		}
	}
	free(stack);
	ts_tree_delete(tree);
	ts_parser_delete(parser);
	return buf;
}

void sqlite_free(char *p) { free(p); }
