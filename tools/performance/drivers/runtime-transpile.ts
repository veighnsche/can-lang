// Preparation only: retain the emitted module graph while removing TypeScript.
import {readdir, readFile, mkdir, writeFile} from "node:fs/promises";
import {join, dirname, relative} from "node:path";
const [source, output] = process.argv.slice(2);
if (!source || !output) throw Error("usage: runtime-transpile SOURCE OUTPUT");
const transpiler = new Bun.Transpiler({loader: "ts", target: "bun"});
let count = 0;
async function walk(directory: string): Promise<void> {
  for (const item of await readdir(directory, {withFileTypes: true})) {
    const path = join(directory, item.name);
    if (item.isDirectory()) await walk(path);
    else if (item.isFile() && item.name.endsWith(".ts")) {
      const target = join(output, relative(source, path).replace(/\.ts$/, ".js"));
      const transformed = transpiler.transformSync(await readFile(path, "utf8"));
      // Emitted/runtime static relative imports and literal dynamic imports keep
      // their layout; package and bun:/node: specifiers remain untouched.
      const javascript = transformed.replace(/((?:from\s*|import\s*(?:\(\s*)?)["']\.\.?\/[^"']+)\.ts(["'])/g, "$1.js$2");
      await mkdir(dirname(target), {recursive: true});
      await writeFile(target, javascript);
      count++;
    }
  }
}
await walk(source);
console.log(JSON.stringify({javascript_modules: count, transformation: "Bun.Transpiler; relative module .ts specifiers rewritten to .js"}));
