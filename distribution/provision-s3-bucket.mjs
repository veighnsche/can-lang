// Idempotent S3 bucket ensure for H03 provisioning. Reads the standard
// CAN_TEST_S3_* env names, signs a SigV4 PUT Bucket, tolerates
// BucketAlreadyOwnedByYou. No dependencies beyond bun/node:crypto.
import { createHmac, createHash } from "node:crypto";
const endpoint = (process.env.CAN_TEST_S3_ENDPOINT ?? "").replace(/\/$/, "");
const region = process.env.CAN_TEST_S3_REGION ?? "";
const access = process.env.CAN_TEST_S3_ACCESS_KEY ?? "";
const secret = process.env.CAN_TEST_S3_SECRET_KEY ?? "";
const bucket = process.env.CAN_TEST_S3_BUCKET ?? "";
if (!endpoint || !region || !access || !secret || !bucket) {
  console.error("set CAN_TEST_S3_ENDPOINT/REGION/BUCKET/ACCESS_KEY/SECRET_KEY");
  process.exit(1);
}
const hmac = (key, data) => createHmac("sha256", key).update(data).digest();
const url = new URL(endpoint + "/" + bucket);
const amz = new Date().toISOString().replace(/[:-]|\.\d{3}/g, "");
const date = amz.slice(0, 8);
const payloadHash = createHash("sha256").update("").digest("hex");
const headers = { host: url.host, "x-amz-content-sha256": payloadHash, "x-amz-date": amz };
const signed = Object.keys(headers).sort();
const canonical =
  `PUT\n${url.pathname}\n\n` +
  signed.map((k) => `${k}:${headers[k]}\n`).join("") +
  `\n${signed.join(";")}\n${payloadHash}`;
const scope = `${date}/${region}/s3/aws4_request`;
const toSign = `AWS4-HMAC-SHA256\n${amz}\n${scope}\n${createHash("sha256").update(canonical).digest("hex")}`;
const k = hmac(hmac(hmac(hmac("AWS4" + secret, date), region), "s3"), "aws4_request");
const sig = createHmac("sha256", k).update(toSign).digest("hex");
const res = await fetch(url, {
  method: "PUT",
  headers: {
    ...headers,
    Authorization: `AWS4-HMAC-SHA256 Credential=${access}/${scope}, SignedHeaders=${signed.join(";")}, Signature=${sig}`,
  },
});
const text = await res.text();
if (res.status === 200 || text.includes("BucketAlreadyOwnedByYou")) {
  console.log(`bucket ok: ${bucket}`);
} else {
  console.error(`mkbucket failed: ${res.status} ${text.slice(0, 200)}`);
  process.exit(1);
}
