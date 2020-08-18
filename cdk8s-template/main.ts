import { App } from 'cdk8s';
import MyChart, { Config } from "./chart";
import { Application } from "./application";
import { readFileSync } from 'fs';

const readStdinAll = async () => {
  process.stdin.setEncoding('utf8');
  let input = '';
  for await (const chunk of process.stdin)
    input += chunk;

  return input;
};

(async () => {
  const prop: Application = JSON.parse(await readStdinAll());
  const config: Config = JSON.parse(readFileSync(process.env["CONFIG_PATH"] ?? "./config.json").toString());

  const app = new App();
  const array = new MyChart(app, `modoki-${prop.metadata.name}`, {app: prop, config}).toJson();

  console.log(JSON.stringify(array));
})();

