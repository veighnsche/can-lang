// S3 integration driver (harness side): proves the provisioned bucket
// is reachable and stages the binary payload under the test prefix.
// Usage: s3-driver.ts setup <payloadPath> <prefix>
import { S3Client } from "bun";

const endpoint = process.env.CAN_TEST_S3_ENDPOINT;
const region = process.env.CAN_TEST_S3_REGION;
const bucket = process.env.CAN_TEST_S3_BUCKET;
const accessKeyId = process.env.CAN_TEST_S3_ACCESS_KEY;
const secretAccessKey = process.env.CAN_TEST_S3_SECRET_KEY;
const [command, payloadPath, prefix] = process.argv.slice(2);
if (command !== "setup" || !payloadPath || !prefix) {
  console.error("usage: s3-driver.ts setup <payloadPath> <prefix>");
  process.exit(2);
}
for (const [name, value] of [["CAN_TEST_S3_ENDPOINT", endpoint], ["CAN_TEST_S3_REGION", region], ["CAN_TEST_S3_BUCKET", bucket], ["CAN_TEST_S3_ACCESS_KEY", accessKeyId], ["CAN_TEST_S3_SECRET_KEY", secretAccessKey]] as const) {
  if (!value) {
    console.error(`missing ${name}`);
    process.exit(2);
  }
}
const client = new S3Client({ endpoint: endpoint!, region: region!, bucket: bucket!, accessKeyId: accessKeyId!, secretAccessKey: secretAccessKey! });
await client.list({ prefix, maxKeys: 1 });
const payload = await Bun.file(payloadPath).bytes();
const staged = `${prefix}payload.bin`;
await client.file(staged).write(payload, { type: "application/octet-stream" });
const stat = await client.file(staged).stat();
// Credentials never enter the report: only the host, bucket and sizes.
console.log(JSON.stringify({ host: new URL(endpoint!).host, bucket, staged, size: stat.size }));
