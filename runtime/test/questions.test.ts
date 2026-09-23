import { test, expect } from "bun:test";
import {
  validateQuestions,
  encodeQuestions,
  decodeAnswers,
  QuestionIssue,
  AnswerIssue,
  type QuestionDescriptor,
  type ChoiceDescriptor,
  type ScoreDescriptor,
} from "../ai/questions.ts";
import { record } from "../data.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
const origin = { source: "test:questions", start: 0, end: 0, invocation: [] };
const noul: QuestionDescriptor = {
  kind: "noul",
  instructions: "Accept?",
  trueDescription: "Yes",
  falseDescription: "No",
  minimum: 0.5,
};
const choice: ChoiceDescriptor = {
  kind: "choice",
  instructions: "Pick",
  options: [
    { key: "later", description: "First" },
    { key: "__proto__", description: "Second" },
  ],
  minimum: 0.6,
};
const score: ScoreDescriptor = {
  kind: "score",
  instructions: "Rate",
  levels: ["Low", "Middle", "High"],
  minimum: 0.2,
};
const batch = [noul, choice, score];
const schema = {
  root: "state",
  nodes: [
    { identity: "state", kind: "record", name: "state", fields: [{ name: "amount", type: "int" }] },
    { identity: "int", kind: "primitive", name: "int" },
  ],
};
const state = record("state", [["amount", 9007199254740993n]]);
const wire = () =>
  JSON.parse(
    '{"model":"resolved","answers":{"q0":{"type":"noul","noul":0.75},"q1":{"type":"choice","choice":"__proto__","confidence":0.01,"probabilities":{"later":0.5,"__proto__":0.5}},"q2":{"type":"score","score":1.4,"confidence":0.8,"probabilities":{"0":0.2,"1":0.2,"2":0.6},"legend":{"0":"Low","1":"Middle","2":"High"}}}}',
  );
const decode = (data: unknown, questions: readonly QuestionDescriptor[] = batch) =>
  decodeAnswers(ownBytes(new TextEncoder().encode(JSON.stringify(data))), questions, 8192);
function badQuestion(question: QuestionDescriptor, reason: QuestionIssue["reason"]) {
  try {
    validateQuestions([noul, question]);
    throw Error("accepted question");
  } catch (cause) {
    expect(cause).toBeInstanceOf(QuestionIssue);
    expect((cause as QuestionIssue).question).toBe("q1");
    expect((cause as QuestionIssue).reason).toBe(reason);
  }
}
function badAnswer(change: (data: any) => void, question: string, reason: AnswerIssue["reason"]) {
  const data = wire();
  change(data);
  try {
    decode(data);
    throw Error("accepted answer");
  } catch (cause) {
    expect(cause).toBeInstanceOf(AnswerIssue);
    expect((cause as AnswerIssue).question).toBe(question);
    expect((cause as AnswerIssue).reason).toBe(reason);
  }
}
test("mixed request uses native exact JSON with local policies omitted and hostile keys intact", () => {
  const encoded = new TextDecoder().decode(
    copyBytes(encodeQuestions("jev-latest", schema, state, batch, 8192), origin),
  );
  expect(encoded).toContain('"amount":9007199254740993');
  const parsed = JSON.parse(encoded);
  expect(Object.keys(parsed.questions)).toEqual(["q0", "q1", "q2"]);
  expect(parsed.questions.q0).toEqual({
    type: "noul",
    instructions: "Accept?",
    criteria: { true: "Yes", false: "No" },
  });
  expect(Object.keys(parsed.questions.q1.criteria)).toEqual(["later", "__proto__"]);
  expect(parsed.questions.q1.criteria.__proto__).toBe("Second");
  expect(parsed.questions.q2).toEqual({
    type: "score",
    instructions: "Rate",
    criteria: ["Low", "Middle", "High"],
  });
  expect(encoded).not.toContain('"minimum"');
});
test("Choice and Score enforce finite counts, exact scalar keys, descriptions and thresholds", () => {
  for (const count of [0, 1, 256])
    badQuestion(
      {
        ...choice,
        options: Array.from({ length: count }, (_, i) => ({
          key: String(i),
          description: "Option",
        })),
      },
      "option_count",
    );
  for (const count of [0, 1, 11])
    badQuestion({ ...score, levels: Array(count).fill("Level") }, "level_count");
  validateQuestions([
    {
      ...choice,
      options: Array.from({ length: 255 }, (_, i) => ({ key: String(i), description: "Option" })),
    },
    { ...score, levels: Array(10).fill("Level") },
  ]);
  for (const key of ["", "\ud800", "x".repeat(257), "😀".repeat(65)])
    badQuestion(
      { ...choice, options: [{ key, description: "Option" }, choice.options[0]] },
      "option_key",
    );
  validateQuestions([
    {
      ...choice,
      options: [
        { key: "😀".repeat(64), description: "Option" },
        { key: " ", description: "Space" },
      ],
    },
  ]);
  badQuestion({ ...choice, options: [choice.options[0], choice.options[0]] }, "duplicate_option");
  badQuestion(
    { ...choice, options: [choice.options[0], { key: "other", description: "\u00a0" }] },
    "criterion",
  );
  badQuestion({ ...score, levels: ["Low", "\udfff"] }, "criterion");
  for (const kind of [choice, score]) {
    badQuestion({ ...kind, instructions: "" }, "instructions");
    for (const minimum of [-1, 2, NaN, Infinity]) badQuestion({ ...kind, minimum }, "threshold");
    validateQuestions([{ ...kind, minimum: undefined }]);
  }
});
test("mixed answers preserve provider tie choice, confidence, score and distribution", () => {
  const answers = decode(wire());
  expect(Object.isFrozen(answers)).toBe(true);
  expect(answers[0]).toEqual({ kind: "noul", probability: 0.75 });
  expect(answers[1]).toEqual({
    kind: "choice",
    choice: "__proto__",
    confidence: 0.01,
    probabilities: [0.5, 0.5],
  });
  expect(answers[2]).toEqual({
    kind: "score",
    score: 1.4,
    confidence: 0.8,
    probabilities: [0.2, 0.2, 0.6],
  });
  for (const answer of answers) {
    expect(Object.isFrozen(answer)).toBe(true);
    if (answer.kind !== "noul") expect(Object.isFrozen(answer.probabilities)).toBe(true);
  }
  const data = wire();
  data.answers.q1.probabilities.later = 0.5000004;
  data.answers.q1.choice = "later";
  data.answers.q2.score = 1.400001;
  const accepted = decode(data);
  expect(accepted[1].kind === "choice" && accepted[1].probabilities[0]).toBe(0.5000004);
  expect(accepted[2].kind === "score" && accepted[2].score).toBe(1.400001);
});
test("full answer admission checks IDs before any payload and every answer before exposure", () => {
  badAnswer(
    (data) => {
      data.answers.extra = {};
      data.answers.q0 = {};
    },
    "",
    "question_ids",
  );
  badAnswer(
    (data) => {
      data.answers.q1.type = "score";
    },
    "q1",
    "answer_type",
  );
  for (const key of ["choice", "confidence", "probabilities"])
    badAnswer(
      (data) => {
        delete data.answers.q1[key];
      },
      "q1",
      "missing_value",
    );
  for (const key of ["score", "confidence", "probabilities", "legend"])
    badAnswer(
      (data) => {
        delete data.answers.q2[key];
      },
      "q2",
      "missing_value",
    );
  for (const value of [null, "0.5", -0.1, 1.1])
    badAnswer(
      (data) => {
        data.answers.q1.confidence = value;
      },
      "q1",
      "confidence",
    );
  badAnswer(
    (data) => {
      data.answers.q1.probabilities = { later: 1 };
    },
    "q1",
    "distribution_keys",
  );
  badAnswer(
    (data) => {
      data.answers.q2.probabilities = { "0": 1, "1": 0, "02": 0 };
    },
    "q2",
    "distribution_keys",
  );
  badAnswer(
    (data) => {
      data.answers.q1.probabilities.later = -0.1;
    },
    "q1",
    "probability",
  );
  badAnswer(
    (data) => {
      data.answers.q1.probabilities.later = 0.4;
    },
    "q1",
    "distribution_sum",
  );
  for (const value of [null, 1, "missing"])
    badAnswer(
      (data) => {
        data.answers.q1.choice = value;
      },
      "q1",
      "selected_key",
    );
  badAnswer(
    (data) => {
      data.answers.q1.probabilities.later = 0.6;
      data.answers.q1.probabilities.__proto__ = 0.4;
    },
    "q1",
    "selected_probability",
  );
  for (const value of [null, "1.4", -1, 3, 1.40001])
    badAnswer(
      (data) => {
        data.answers.q2.score = value;
      },
      "q2",
      "score",
    );
  badAnswer(
    (data) => {
      data.answers.q2.legend["2"] = "Wrong";
    },
    "q2",
    "legend",
  );
  badAnswer(
    (data) => {
      data.answers.q2.legend.extra = "Extra";
    },
    "q2",
    "legend",
  );
  badAnswer(
    (data) => {
      data.answers.q2.probabilities["0"] = 1.1;
    },
    "q2",
    "probability",
  );
});
