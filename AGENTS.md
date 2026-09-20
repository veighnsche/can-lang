There are zero external users. Implementing with backwards compatibility immediately results in technical debt. Maintaining syntax, ABI, old source spellings, generated TSlayouts, or old goldens is not a product requirement.

Compile Can operations to equivalent native JavaScript/Bun operations in generated TypeScript instead of reimplementing them in Can or generated helpers, adding only adapters needed to preserve Can's contracts and immutability.

You can ask Jev (TypeSafe AI) SystemOne for design decisions, but you have to give it all the relevant context yourself. Jev CANNOT do research before it answers, it's not capable of doing that. Jev is a very advanced classifier model, NOT an LLM. For more precision ask Jev the same question with the same content just worded completely different to even out eventual hidden biases.
